package ra

import "testing"

func TestRequestOutcome(t *testing.T) {
	cases := []struct {
		name   string
		status int
		route  string
		raErr  string
		want   string
	}{
		{name: "ok", status: 200, route: "/auth", want: "ok"},
		{name: "auth denied", status: 403, route: "/auth", want: "auth_denied"},
		{name: "body validation", status: 422, route: "/topics/*proxyPath", raErr: "invalid", want: "body_validation_failed"},
		{name: "proxy upstream", status: 502, route: "/topics/*proxyPath", want: "proxy_upstream_error"},
		{name: "generic error", status: 500, route: "/auth", want: "client_or_server_error"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := requestOutcome(tc.status, tc.route, tc.raErr)
			if got != tc.want {
				t.Fatalf("requestOutcome()=%q want=%q", got, tc.want)
			}
		})
	}
}
