package strconv

import "testing"

func TestToSnakeCase(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"AddContract", "add_contract"},
		{"Add", "add"},
		{"HTTPRequest", "http_request"},
		{"GetURLByID", "get_url_by_id"},
		{"GetIP", "get_ip"},
		{"AMethod", "a_method"},
		{"JSONToXML", "json_to_xml"},
		{"OAuth2Login", "o_auth2_login"}, // This is a tricky case, but we can live with "o_auth2_login" for simplicity.
		{"S3Upload", "s3_upload"},
		{"Ping", "ping"},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			got := ToSnakeCase(tc.input)
			if got != tc.expected {
				t.Errorf("ToSnakeCase(%q) = %q, want %q", tc.input, got, tc.expected)
			}
		})
	}
}
