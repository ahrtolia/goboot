package nacos

//
//import (
//	"github.com/nacos-group/nacos-sdk-go/v2/clients/config_client"
//	"github.com/nacos-group/nacos-sdk-go/v2/vo"
//	"github.com/spf13/viper"
//	"reflect"
//	"testing"
//)
//
//func Test_initConfigClient(t *testing.T) {
//	tests := []struct {
//		name    string
//		want    config_client.IConfigClient
//		wantErr bool
//	}{
//		{
//			name:    "test",
//			want:    nil,
//			wantErr: false,
//		},
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			configClient, err := initConfigClient(viper.New())
//			if (err != nil) != tt.wantErr {
//				t.Errorf("initConfigClient() error = %v, wantErr %v", err, tt.wantErr)
//				return
//			}
//			if !reflect.DeepEqual(configClient, tt.want) {
//				t.Errorf("initConfigClient() got = %v, want %v", configClient, tt.want)
//			}
//
//			//got.
//			_, err = configClient.PublishConfig(vo.ConfigParam{
//				DataId:  "dataId",
//				Group:   "group",
//				Content: "hello world!222222",
//				AppName: "app",
//			})
//
//			if err != nil {
//				t.Error(err)
//			}
//		})
//	}
//}
