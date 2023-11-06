package internal_test

import (
	"github.com/stretchr/testify/assert"

	"testing"

	"github.com/apecloud/dataprotection-wal-g/internal"
	"github.com/spf13/viper"
)

func TestDtoSerializer_NewDtoSerializer(t *testing.T) {
	tests := []struct {
		name                       string
		serializerTypeSettingValue string
		expectedDto                internal.DtoSerializer
		expectedErrText            string
	}{
		{
			name:                       "ReguralJSON_if_json_default",
			serializerTypeSettingValue: "json_default",
			expectedDto:                internal.RegularJSON{},
			expectedErrText:            "",
		},
		{
			name:                       "StreamedJSON_if_json_streamed",
			serializerTypeSettingValue: "json_streamed",
			expectedDto:                internal.StreamedJSON{},
			expectedErrText:            "",
		},
		{
			name:                       "error_if_unknown_type",
			serializerTypeSettingValue: "ff",
			expectedDto:                nil,
			expectedErrText:            "undefined dto serializer type: ff",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			viper.Set(internal.SerializerTypeSetting, tt.serializerTypeSettingValue)

			dto, err := internal.NewDtoSerializer()

			if tt.expectedErrText == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tt.expectedErrText, "Errors not same")
			}
			assert.Equalf(t, tt.expectedDto, dto, "Expected different dto")
		})
	}
}
