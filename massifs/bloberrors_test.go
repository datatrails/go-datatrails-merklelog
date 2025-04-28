package massifs

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/stretchr/testify/assert"
)

func TestIsRateLimiting(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		minWait        time.Duration
		maxWait        time.Duration
		expectedWait   time.Duration
		expectedResult bool
	}{
		{
			name: "429 with Retry-After header as date",
			err: &azcore.ResponseError{
				StatusCode: http.StatusTooManyRequests,
				RawResponse: &http.Response{
					Header: http.Header{
						"Retry-After": []string{time.Now().Add(5 * time.Minute).UTC().Format(http.TimeFormat)},
					},
				},
			},
			minWait:        4*time.Minute + 59*time.Second,
			maxWait:        6 * time.Minute, // remove the max wait if this turnes out to be flaky but it should be fine
			expectedResult: true,
		},
		{
			name:           "nil error",
			err:            nil,
			expectedWait:   0,
			expectedResult: false,
		},
		{
			name:           "non-response error",
			err:            errors.New("some error"),
			expectedWait:   0,
			expectedResult: false,
		},
		{
			name: "non-429 status code",
			err: &azcore.ResponseError{
				StatusCode: http.StatusInternalServerError,
			},
			expectedWait:   0,
			expectedResult: false,
		},
		{
			name: "429 without Retry-After header",
			err: &azcore.ResponseError{
				StatusCode: http.StatusTooManyRequests,
				RawResponse: &http.Response{
					Header: http.Header{},
				},
			},
			expectedWait:   0,
			expectedResult: true,
		},
		{
			name: "429 with Retry-After header in seconds",
			err: &azcore.ResponseError{
				StatusCode: http.StatusTooManyRequests,
				RawResponse: &http.Response{
					Header: http.Header{
						"Retry-After": []string{"10"},
					},
				},
			},
			expectedWait:   10 * time.Second,
			expectedResult: true,
		},
		{
			name: "429 with invalid Retry-After header",
			err: &azcore.ResponseError{
				StatusCode: http.StatusTooManyRequests,
				RawResponse: &http.Response{
					Header: http.Header{
						"Retry-After": []string{"invalid"},
					},
				},
			},
			expectedWait:   0,
			expectedResult: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			wait, result := IsRateLimiting(tt.err)
			// Note: we use a range check here because parsing the Retry-After header
			// uses time.Until, which then calls time.Now, in the case where the header is a date.
			// this means this test would be flaky. We could mock time.Now but that would be a lot of work
			if tt.minWait > 0 {
				assert.GreaterOrEqual(t, wait, tt.minWait)
			}
			if tt.maxWait > 0 {
				assert.LessOrEqual(t, wait, tt.maxWait)
			}
			if tt.expectedWait > 0 || (tt.minWait == 0 && tt.maxWait == 0) {
				assert.Equal(t, tt.expectedWait, wait)
			}
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}
