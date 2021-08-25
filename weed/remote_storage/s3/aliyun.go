package s3

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/chrislusf/seaweedfs/weed/pb/filer_pb"
	"github.com/chrislusf/seaweedfs/weed/remote_storage"
	"github.com/chrislusf/seaweedfs/weed/util"
	"os"
)

func init() {
	remote_storage.RemoteStorageClientMakers["aliyun"] = new(AliyunRemoteStorageMaker)
}

type AliyunRemoteStorageMaker struct{}

func (s AliyunRemoteStorageMaker) Make(conf *filer_pb.RemoteConf) (remote_storage.RemoteStorageClient, error) {
	client := &s3RemoteStorageClient{
		conf: conf,
	}
	accessKey := util.Nvl(conf.AliyunAccessKey, os.Getenv("ALICLOUD_ACCESS_KEY_ID"))
	secretKey := util.Nvl(conf.AliyunSecretKey, os.Getenv("ALICLOUD_ACCESS_KEY_SECRET"))

	config := &aws.Config{
		Endpoint:         aws.String(conf.AliyunEndpoint),
		Region:           aws.String(conf.AliyunRegion),
		S3ForcePathStyle: aws.Bool(false),
	}
	if accessKey != "" && secretKey != "" {
		config.Credentials = credentials.NewStaticCredentials(accessKey, secretKey, "")
	}

	sess, err := session.NewSession(config)
	if err != nil {
		return nil, fmt.Errorf("create aliyun session: %v", err)
	}
	client.conn = s3.New(sess)
	return client, nil
}
