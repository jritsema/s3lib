# s3lib

Just another go library for s3 that makes it a little easier to work with.


### API

- PutObject
- GetObject
- DeleteObject
- List
- PutContent
- GetString
- KeyExists
- BucketKeyExists
- UploadDirectory
- UploadFile
- DownloadFile
- GetPresignedURL


### usage

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

*Note that s3 is eventually consistent and should generally not be used as a database.