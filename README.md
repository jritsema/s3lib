# s3lib

Just another go library for s3 that makes it a little easier to work with.

### API

#### Utilities

- NewClient
- NewClientWithSession
- NewClientWithConfig
- BucketKeyExists
- DownloadFile
- UploadDirectory
- UploadFile
- GetPresignedURL

#### Object Persistence

- GetObject
- GetString
- PutObject
- DeleteObject
- PutContent
- List
- KeyExists


### Usage

```
go get github.com/jritsema/s3lib
```

```go
s3, err := s3lib.NewClient("my-bucket", "us-east-1")
check(err)

//save an object
obj := &myType{version: "1.0"}
key := "mykey"
err = s3.PutObject(key, obj)
check(err)

//fetch
_, err = s3.GetObject(key, obj)
check(err)

//update
obj.version = "2.0"
err = s3.PutObject(key, obj)
check(err)

//delete
err = s3.DeleteObject(key)
check(err)
```
