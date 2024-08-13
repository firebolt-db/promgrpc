package promgrpc_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/stats"
	"google.golang.org/grpc/status"

	"github.com/piotrkowalczuk/promgrpc/v4"
)

func TestNewClientResponsesTotalStatsHandler(t *testing.T) {
	ctx := promgrpc.DynamicLabelValuesToCtx(context.Background(), map[string]string{dynamicLabel: dynamicLabelValue})
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)

	defer cancel()
	collectorOpts, statsHandlerOpts := promgrpc.OptionsSplit(
		promgrpc.CollectorStatsHandlerWithDynamicLabels([]string{dynamicLabel}),
	)
	h := promgrpc.NewStatsHandler(
		promgrpc.NewClientResponsesTotalStatsHandler(
			promgrpc.NewClientResponsesTotalCounterVec(collectorOpts...),
			statsHandlerOpts...,
		))
	ctx = h.TagRPC(ctx, &stats.RPCTagInfo{
		FullMethodName: "/service/Method",
		FailFast:       true,
	})
	h.HandleRPC(ctx, &stats.OutHeader{
		Client: true,
		Header: metadata.MD{"user-agent": []string{"fake-user-agent"}},
	})
	h.HandleRPC(ctx, &stats.End{
		Client: true,
		Error:  status.Error(codes.Aborted, "aborted"),
	})
	h.HandleRPC(ctx, &stats.End{
		Client: true,
	})
	h.HandleRPC(ctx, &stats.End{
		Client: false,
	})

	const metadata = `
		# HELP grpc_client_responses_received_total TODO
        # TYPE grpc_client_responses_received_total counter
	`
	expected := fmt.Sprintf(`
		grpc_client_responses_received_total{%[1]s="%[2]s",grpc_client_user_agent="fake-user-agent",grpc_code="Aborted",grpc_is_fail_fast="true",grpc_method="Method",grpc_service="service"} 1
        grpc_client_responses_received_total{%[1]s="%[2]s",grpc_client_user_agent="fake-user-agent",grpc_code="OK",grpc_is_fail_fast="true",grpc_method="Method",grpc_service="service"} 1
	`, dynamicLabel, dynamicLabelValue)

	if err := testutil.CollectAndCompare(h, strings.NewReader(metadata+expected), "grpc_client_responses_received_total"); err != nil {
		t.Fatal(err)
	}
}
