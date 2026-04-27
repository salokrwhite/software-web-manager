本文介绍如何通过简单上传方法将本地文件快速上传到OSS，此方法操作简便，适合快速将本地文件上传到云端存储。

## **注意事项**

-   本文示例代码以华东1（杭州）的地域ID`cn-hangzhou`为例，默认使用外网Endpoint，如果您希望通过与OSS同地域的其他阿里云产品访问OSS，请使用内网Endpoint。关于OSS支持的Region与Endpoint的对应关系，请参见[OSS地域和访问域名](https://help.aliyun.com/zh/oss/user-guide/regions-and-endpoints#concept-zt4-cvy-5db)。
    
-   本文以从环境变量读取访问凭证为例。如何配置访问凭证，请参见[配置访问凭证](https://help.aliyun.com/zh/oss/developer-reference/configure-access-credentials-by-using-oss-sdk-for-go-v2)。
    

## **权限说明**

阿里云账号默认拥有全部权限。阿里云账号下的RAM用户或RAM角色默认没有任何权限，需要阿里云账号或账号管理员通过[RAM Policy](https://help.aliyun.com/zh/oss/ram-policy-overview/)或[Bucket Policy](https://help.aliyun.com/zh/oss/user-guide/oss-bucket-policy/)授予操作权限。

| **API**                | **Action**                                                   | **说明**     |
| ---------------------- | ------------------------------------------------------------ | ------------ |
| PutObject              | `oss:PutObject`                                              | 上传Object。 |
| `oss:PutObjectTagging` | 上传Object时，如果通过x-oss-tagging指定Object的标签，则需要此操作的权限。 |              |
| `kms:GenerateDataKey`  | 上传Object时，如果Object的元数据包含X-Oss-Server-Side-Encryption: KMS，则需要这两个操作的权限。 |              |
| `kms:Decrypt`          |                                                              |              |

## **方法定义**

```
func (c *Client) PutObject(ctx context.Context, request *PutObjectRequest, optFns ...func(*Options)) (*PutObjectResult, error)

func (c *Client) PutObjectFromFile(ctx context.Context, request *PutObjectRequest, filePath string, optFns ...func(*Options)) (*PutObjectResult, error)
```

| 接口名                   | 说明                                                         |
| ------------------------ | ------------------------------------------------------------ |
| Client.PutObject         | 简单上传, 最大支持5GiB 支持CRC64数据校验（默认启用） 支持进度条 请求body类型为io.Reader, 当支持io.Seeker类型时，具备失败重传 |
| Client.PutObjectFromFile | 与Client.PutObject接口能力一致 请求body数据来源于文件路径    |

#### **请求参数列表**

| 参数名  | 类型                | 说明                                                         |
| ------- | ------------------- | ------------------------------------------------------------ |
| ctx     | context.Context     | 请求的上下文，可以用来设置请求的总时限                       |
| request | \\*PutObjectRequest | 设置具体接口的请求参数，例如设置对象的访问控制方式（Acl）、禁止覆盖写（ForbidOverwrite）、自定义元数据（Metadata）等，具体请参见[PutObjectRequest](https://pkg.go.dev/github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss#PutObjectRequest) |
| optFns  | ...func(\\*Options) | （可选）接口级的配置参数, 具体请参见[Options](https://pkg.go.dev/github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss#Options) |

#### **返回值列表**

| 返回值名 | 类型               | 说明                                                         |
| -------- | ------------------ | ------------------------------------------------------------ |
| result   | \\*PutObjectResult | 接口返回值，当 err 为nil 时有效，具体请参见[PutObjectResult](https://pkg.go.dev/github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss#PutObjectResult) |
| err      | error              | 请求的状态，当请求失败时，err 不为 nil                       |

## **示例代码**

您可以使用以下代码将本地文件上传到目标存储空间。

```
package main

import (
	"context"
	"flag"
	"log"

	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss/credentials"
)

// 定义全局变量
var (
	region     string // 存储区域
	bucketName string // 存储空间名称
	objectName string // 对象名称
)

// init函数用于初始化命令行参数
func init() {
	flag.StringVar(&region, "region", "", "The region in which the bucket is located.")
	flag.StringVar(&bucketName, "bucket", "", "The name of the bucket.")
	flag.StringVar(&objectName, "object", "", "The name of the object.")
}

func main() {
	// 解析命令行参数
	flag.Parse()

	// 检查bucket名称是否为空
	if len(bucketName) == 0 {
		flag.PrintDefaults()
		log.Fatalf("invalid parameters, bucket name required")
	}

	// 检查region是否为空
	if len(region) == 0 {
		flag.PrintDefaults()
		log.Fatalf("invalid parameters, region required")
	}

	// 检查object名称是否为空
	if len(objectName) == 0 {
		flag.PrintDefaults()
		log.Fatalf("invalid parameters, object name required")
	}

	// 加载默认配置并设置凭证提供者和区域
	cfg := oss.LoadDefaultConfig().
		WithCredentialsProvider(credentials.NewEnvironmentVariableCredentialsProvider()).
		WithRegion(region)

	// 创建OSS客户端
	client := oss.NewClient(cfg)

	// 填写要上传的本地文件路径和文件名称，例如 /Users/localpath/exampleobject.txt
	localFile := "/Users/localpath/exampleobject.txt"

	// 创建上传对象的请求
	putRequest := &oss.PutObjectRequest{
		Bucket:       oss.Ptr(bucketName),      // 存储空间名称
		Key:          oss.Ptr(objectName),      // 对象名称
		StorageClass: oss.StorageClassStandard, // 指定对象的存储类型为标准存储
		Acl:          oss.ObjectACLPrivate,     // 指定对象的访问权限为私有访问
		Metadata: map[string]string{
			"yourMetadataKey1": "yourMetadataValue1", // 设置对象的元数据
		},
	}

	// 执行上传对象的请求
	result, err := client.PutObjectFromFile(context.TODO(), putRequest, localFile)
	if err != nil {
		log.Fatalf("failed to put object from file %v", err)
	}

	// 打印上传对象的结果
	log.Printf("put object from file result:%#v\n", result)
}
```

## **常见使用场景**

### **上传字符串**

您可以使用以下代码将字符串上传至目标存储空间。

```
package main

import (
	"context"
	"flag"
	"log"
	"strings"

	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss/credentials"
)

// 定义全局变量
var (
	region     string // 存储区域
	bucketName string // 存储空间名称
	objectName string // 对象名称
)

// init函数用于初始化命令行参数
func init() {
	flag.StringVar(&region, "region", "", "The region in which the bucket is located.")
	flag.StringVar(&bucketName, "bucket", "", "The name of the bucket.")
	flag.StringVar(&objectName, "object", "", "The name of the object.")
}

func main() {
	// 解析命令行参数
	flag.Parse()

	// 检查bucket名称是否为空
	if len(bucketName) == 0 {
		flag.PrintDefaults()
		log.Fatalf("invalid parameters, bucket name required")
	}

	// 检查region是否为空
	if len(region) == 0 {
		flag.PrintDefaults()
		log.Fatalf("invalid parameters, region required")
	}

	// 检查object名称是否为空
	if len(objectName) == 0 {
		flag.PrintDefaults()
		log.Fatalf("invalid parameters, object name required")
	}

	// 定义要上传的字符串内容
	body := strings.NewReader("hi oss")

	// 加载默认配置并设置凭证提供者和区域
	cfg := oss.LoadDefaultConfig().
		WithCredentialsProvider(credentials.NewEnvironmentVariableCredentialsProvider()).
		WithRegion(region)

	// 创建OSS客户端
	client := oss.NewClient(cfg)

	// 创建上传对象的请求
	request := &oss.PutObjectRequest{
		Bucket: oss.Ptr(bucketName), // 存储空间名称
		Key:    oss.Ptr(objectName), // 对象名称
		Body:   body,                // 要上传的字符串内容
	}

	// 发送上传对象的请求
	result, err := client.PutObject(context.TODO(), request)
	if err != nil {
		log.Fatalf("failed to put object %v", err)
	}

	// 打印上传对象的结果
	log.Printf("put object result:%#v\n", result)
}
```

### **上传Byte数组**

您可以使用以下代码将Byte数组上传至目标存储空间。

```
package main

import (
	"bytes"
	"context"
	"flag"
	"log"

	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss/credentials"
)

// 定义全局变量
var (
	region     string // 存储区域
	bucketName string // 存储空间名称
	objectName string // 对象名称
)

// init函数用于初始化命令行参数
func init() {
	flag.StringVar(&region, "region", "", "The region in which the bucket is located.")
	flag.StringVar(&bucketName, "bucket", "", "The name of the bucket.")
	flag.StringVar(&objectName, "object", "", "The name of the object.")
}

func main() {
	// 解析命令行参数
	flag.Parse()

	// 检查bucket名称是否为空
	if len(bucketName) == 0 {
		flag.PrintDefaults()
		log.Fatalf("invalid parameters, bucket name required")
	}

	// 检查region是否为空
	if len(region) == 0 {
		flag.PrintDefaults()
		log.Fatalf("invalid parameters, region required")
	}

	// 检查object名称是否为空
	if len(objectName) == 0 {
		flag.PrintDefaults()
		log.Fatalf("invalid parameters, object name required")
	}

	// 定义要上传的byte数组内容
	body := bytes.NewReader([]byte("yourObjectValueByteArray"))

	// 加载默认配置并设置凭证提供者和区域
	cfg := oss.LoadDefaultConfig().
		WithCredentialsProvider(credentials.NewEnvironmentVariableCredentialsProvider()).
		WithRegion(region)

	// 创建OSS客户端
	client := oss.NewClient(cfg)

	// 创建上传对象的请求
	request := &oss.PutObjectRequest{
		Bucket: oss.Ptr(bucketName), // 存储空间名称
		Key:    oss.Ptr(objectName), // 对象名称
		Body:   body,                // 要上传的字符串内容
	}

	// 发送上传对象的请求
	result, err := client.PutObject(context.TODO(), request)
	if err != nil {
		log.Fatalf("failed to put object %v", err)
	}

	// 打印上传对象的结果
	log.Printf("put object result:%#v\n", result)
}
```

### **上传网络流**

您可以使用以下代码将网络上传至目标存储空间。

```
package main

import (
	"context"
	"flag"
	"io"
	"log"
	"net/http"

	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss/credentials"
)

// 定义全局变量
var (
	region     string // 存储区域
	bucketName string // 存储空间名称
	objectName string // 对象名称
)

// init函数用于初始化命令行参数
func init() {
	flag.StringVar(&region, "region", "", "The region in which the bucket is located.")
	flag.StringVar(&bucketName, "bucket", "", "The name of the bucket.")
	flag.StringVar(&objectName, "object", "", "The name of the object.")
}

func main() {
	// 解析命令行参数
	flag.Parse()

	// 检查bucket名称是否为空
	if len(bucketName) == 0 {
		flag.PrintDefaults()
		log.Fatalf("invalid parameters, bucket name required")
	}

	// 检查region是否为空
	if len(region) == 0 {
		flag.PrintDefaults()
		log.Fatalf("invalid parameters, region required")
	}

	// 检查object名称是否为空
	if len(objectName) == 0 {
		flag.PrintDefaults()
		log.Fatalf("invalid parameters, object name required")
	}

	// 加载默认配置并设置凭证提供者和区域
	cfg := oss.LoadDefaultConfig().
		WithCredentialsProvider(credentials.NewEnvironmentVariableCredentialsProvider()).
		WithRegion(region)

	// 创建OSS客户端
	client := oss.NewClient(cfg)

	// 指定待上传的网络流
	resp, err := http.Get("https://www.aliyun.com/")
	if err != nil {
		log.Fatalf("Failed to fetch URL: %v", err)
	}
	defer resp.Body.Close()

	// 创建上传对象的请求
	request := &oss.PutObjectRequest{
		Bucket: oss.Ptr(bucketName),  // 存储空间名称
		Key:    oss.Ptr(objectName),  // 对象名称
		Body:   io.Reader(resp.Body), // 要上传的网络流内容
	}

	// 发送上传对象的请求
	result, err := client.PutObject(context.TODO(), request)
	if err != nil {
		log.Fatalf("failed to put object %v", err)
	}

	// 打印上传对象的结果
	log.Printf("put object result:%#v\n", result)
}
```

### **上传文件时显示进度**

当您在上传文件时，可以使用进度条实时了解上传进度，避免因为等待时间过长而感到不安或怀疑任务是否卡住。

您可以通过以下代码使用进度条查看上传文件的进度。

```
package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss/credentials"
)

// 定义全局变量
var (
	region     string // 存储区域
	bucketName string // 存储空间名称
	objectName string // 对象名称
)

// init函数用于初始化命令行参数
func init() {
	flag.StringVar(&region, "region", "", "The region in which the bucket is located.")
	flag.StringVar(&bucketName, "bucket", "", "The name of the bucket.")
	flag.StringVar(&objectName, "object", "", "The name of the object.")
}

func main() {
	// 解析命令行参数
	flag.Parse()

	// 检查bucket名称是否为空
	if len(bucketName) == 0 {
		flag.PrintDefaults()
		log.Fatalf("invalid parameters, bucket name required")
	}

	// 检查region是否为空
	if len(region) == 0 {
		flag.PrintDefaults()
		log.Fatalf("invalid parameters, region required")
	}

	// 检查object名称是否为空
	if len(objectName) == 0 {
		flag.PrintDefaults()
		log.Fatalf("invalid parameters, object name required")
	}

	// 加载默认配置并设置凭证提供者和区域
	cfg := oss.LoadDefaultConfig().
		WithCredentialsProvider(credentials.NewEnvironmentVariableCredentialsProvider()).
		WithRegion(region)

	// 创建OSS客户端
	client := oss.NewClient(cfg)

	// 填写要上传的本地文件路径和文件名称，例如 /Users/localpath/exampleobject.txt
	localFile := "/Users/localpath/exampleobject.txt"

	// 创建上传对象的请求
	putRequest := &oss.PutObjectRequest{
		Bucket: oss.Ptr(bucketName), // 存储空间名称
		Key:    oss.Ptr(objectName), // 对象名称
		ProgressFn: func(increment, transferred, total int64) {
			fmt.Printf("increment:%v, transferred:%v, total:%v\n", increment, transferred, total)
		}, // 进度回调函数，用于显示上传进度
	}

	// 发送上传对象的请求
	result, err := client.PutObjectFromFile(context.TODO(), putRequest, localFile)
	if err != nil {
		log.Fatalf("failed to put object from file %v", err)
	}

	// 打印上传对象的结果
	log.Printf("put object from file result:%#v\n", result)
}
```

### **上传文件时设置回调函数**

OSS在完成简单上传（PutObject和PutObjectFromFile）时可以提供回调（Callback）给应用服务器。您只需要在发送给OSS的请求中携带相应的Callback参数，即可实现回调。

您可以使用以下代码在上传文件时设置回调函数。

```
package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"log"
	"strings"

	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss/credentials"
)

// 定义全局变量
var (
	region     string // 存储区域
	bucketName string // 存储空间名称
	objectName string // 对象名称
)

// init函数用于初始化命令行参数
func init() {
	flag.StringVar(&region, "region", "", "The region in which the bucket is located.")
	flag.StringVar(&bucketName, "bucket", "", "The name of the bucket.")
	flag.StringVar(&objectName, "object", "", "The name of the object.")
}

func main() {
	// 解析命令行参数
	flag.Parse()

	// 检查bucket名称是否为空
	if len(bucketName) == 0 {
		flag.PrintDefaults()
		log.Fatalf("invalid parameters, bucket name required")
	}

	// 检查region是否为空
	if len(region) == 0 {
		flag.PrintDefaults()
		log.Fatalf("invalid parameters, region required")
	}

	// 检查object名称是否为空
	if len(objectName) == 0 {
		flag.PrintDefaults()
		log.Fatalf("invalid parameters, object name required")
	}

	// 加载默认配置并设置凭证提供者和区域
	cfg := oss.LoadDefaultConfig().
		WithCredentialsProvider(credentials.NewEnvironmentVariableCredentialsProvider()).
		WithRegion(region)

	// 创建OSS客户端
	client := oss.NewClient(cfg)

	// 定义回调参数
	callbackMap := map[string]string{
		"callbackUrl":      "http://example.com:23450",                                                        // 设置回调服务器的URL，例如https://example.com:23450。
		"callbackBody":     "bucket=${bucket}&object=${object}&size=${size}&my_var_1=${x:my_var1}&my_var_2=${x:my_var2}", // 设置回调请求体。
		"callbackBodyType": "application/x-www-form-urlencoded",                                                          //设置回调请求体类型。
	}

	// 将回调参数转换为JSON并进行Base64编码，以便将其作为回调参数传递
	callbackStr, err := json.Marshal(callbackMap)
	if err != nil {
		log.Fatalf("failed to marshal callback map: %v", err)
	}
	callbackBase64 := base64.StdEncoding.EncodeToString(callbackStr)

	callbackVarMap := map[string]string{}
	callbackVarMap["x:my_var1"] = "thi is var 1"
	callbackVarMap["x:my_var2"] = "thi is var 2"
	callbackVarStr, err := json.Marshal(callbackVarMap)
	if err != nil {
		log.Fatalf("failed to marshal callback var: %v", err)
	}
	callbackVarBase64 := base64.StdEncoding.EncodeToString(callbackVarStr)
	// 定义要上传的字符串内容
	body := strings.NewReader("Hello, OSS!") // 替换为要上传的字符串内容

	// 创建上传对象的请求
	request := &oss.PutObjectRequest{
		Bucket:      oss.Ptr(bucketName),     // 存储空间名称
		Key:         oss.Ptr(objectName),     // 对象名称
		Body:        body,                    // 对象内容
		Callback:    oss.Ptr(callbackBase64), // 填写回调参数
		CallbackVar: oss.Ptr(callbackVarBase64),
	}

	// 发送上传对象的请求
	result, err := client.PutObject(context.TODO(), request)
	if err != nil {
		log.Fatalf("failed to put object %v", err)
	}

	// 打印上传对象的结果
	log.Printf("put object result:%#v\n", result)
}
```

## 相关文档

-   关于简单上传的完整示例代码，请参见GitHub示例[put\_object.go](https://github.com/aliyun/alibabacloud-oss-go-sdk-v2/blob/master/sample/put_object.go)和[put\_object\_from\_file.go](https://github.com/aliyun/alibabacloud-oss-go-sdk-v2/blob/master/sample/put_object_form_file.go)。
    
-   关于简单上传的API接口说明，请参见[PutObject](https://pkg.go.dev/github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss#Client.PutObject)和[PutObjectFromFile](https://pkg.go.dev/github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss#Client.PutObjectFromFile)。