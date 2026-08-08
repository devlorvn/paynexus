package interceptors

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"strconv"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	HeaderAPIKey    = "x-nexus-api-key"
	HeaderSignature = "x-nexus-signature"
	HeaderTimestamp = "x-nexus-timestamp"
)

func UnaryAuthInterceptor(logger *zap.Logger, ignoreMethods []string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		for _, m := range ignoreMethods {
			if info.FullMethod == m {
				return handler(ctx, req)
			}
		}

		md, ok := metadata.FromIncomingContext(ctx)

		if !ok {
			return nil, status.Error(codes.Unauthenticated, "Invalid gRPC metadata header")
		}

		apiKeys := md.Get((HeaderAPIKey))
		signatures := md.Get((HeaderSignature))
		timestamps := md.Get((HeaderTimestamp))

		if len(apiKeys) == 0 || len(signatures) == 0 || len(timestamps) == 0 {
			return nil, status.Error(codes.Unauthenticated, "Invalid gRPC metadata header")
		}

		apiKey := apiKeys[0]
		signature := signatures[0]
		timestamp := timestamps[0]

		ts, err := strconv.ParseInt(timestamp, 10, 64)
		if err != nil || math.Abs(float64(time.Now().Unix()-ts)) > 300 {
			logger.Warn("Request expired or relayed attack", zap.String("timestamp", timestamp))
			return nil, status.Error(codes.Unauthenticated, "Invalid timestamp")
		}

		mockSecret := "sk_..."

		expectedSignature := calculateHMAC(fmt.Sprintf("%s.%d", apiKey, ts), mockSecret)

		if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
			logger.Warn("Invalid signature", zap.String("signature", signature), zap.String("expected", expectedSignature), zap.String("apiKey", apiKey))
			return nil, status.Error(codes.Unauthenticated, "Invalid signature")
		}

		ctx = InjectMerchantContext(ctx, 1, info.FullMethod, apiKey)

		return handler(ctx, req)
	}
}

func calculateHMAC(data, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}
