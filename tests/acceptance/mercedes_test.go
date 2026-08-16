package acceptance

import "testing"

func TestMercedesModelLoadsWithSoCDecoder(t *testing.T) {
	startMateConfigured(t, `env:
  MERCEDES_EMAIL: test@example.com
  MERCEDES_PASSWORD: test-password
  MERCEDES_VIN: TESTVIN
  MERCEDES_REGION: EU`, "example/mercedes")
}
