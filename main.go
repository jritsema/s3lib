package s3lib

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

//Client is an s3 client
type Client struct {
	session *session.Session
	svc     *s3.S3
	Bucket  string
	Region  string
}

//NewClient creates a new client
func NewClient(bucket string, region string) (*Client, error) {

	sess, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	if err != nil {
		return nil, err
	}

	return NewClientWithSession(sess, bucket), nil
}

//NewClientWithSession creates a new client based on a session
func NewClientWithSession(s *session.Session, bucket string) *Client {
	return &Client{
		session: s,
		svc:     s3.New(s),
		Bucket:  bucket,
		Region:  *s.Config.Region,
	}
}

//GetObject returns an object from JSON for a key.
//Return false if key is not found.
func (c *Client) GetObject(key string, value interface{}) (bool, error) {

	input := &s3.GetObjectInput{
		Bucket: aws.String(c.Bucket),
		Key:    aws.String(key),
	}

	result, err := c.svc.GetObject(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() == "NoSuchKey" {
				return false, nil
			}
		}
		return false, err
	}

	defer result.Body.Close()
	decoder := json.NewDecoder(result.Body)
	err = decoder.Decode(value)
	if err != nil {
		return false, err
	}

	return true, nil
}

//GetString returns a string representation of a key
func (c *Client) GetString(key string) (string, error) {

	input := &s3.GetObjectInput{
		Bucket: aws.String(c.Bucket),
		Key:    aws.String(key),
	}

	result, err := c.svc.GetObject(input)
	if err != nil {
		return "", err
	}
	defer result.Body.Close()
	bits, err := ioutil.ReadAll(result.Body)
	if err != nil {
		return "", err
	}

	return string(bits), nil
}

//PutObject marshals an object to JSON and writes it to a key
func (c *Client) PutObject(key string, value interface{}) error {

	input := s3.PutObjectInput{
		Bucket: aws.String(c.Bucket),
		Key:    aws.String(key),
	}

	if value != nil {
		json, _ := json.MarshalIndent(value, "", "  ")
		input.Body = bytes.NewReader(json)
		input.ContentType = aws.String("application/json")
	}

	_, err := c.svc.PutObject(&input)
	if err != nil {
		return err
	}

	return nil
}

//DeleteObject deletes an object from S3
func (c *Client) DeleteObject(key string) error {

	input := s3.DeleteObjectInput{
		Bucket: aws.String(c.Bucket),
		Key:    aws.String(key),
	}

	_, err := c.svc.DeleteObject(&input)
	if err != nil {
		return err
	}

	return nil
}

//PutContent writes content to a key
func (c *Client) PutContent(key string, body io.ReadSeeker, contentType string) error {

	input := s3.PutObjectInput{
		Bucket:      aws.String(c.Bucket),
		Key:         aws.String(key),
		Body:        body,
		ContentType: &contentType,
	}

	_, err := c.svc.PutObject(&input)
	if err != nil {
		return err
	}

	return nil
}

//List lists bucket keys with an optional prefix filter
func (c *Client) List(prefix string) (*s3.ListObjectsV2Output, error) {

	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(c.Bucket),
	}
	if prefix != "" {
		input.Prefix = aws.String(prefix)
	}

	output, err := c.svc.ListObjectsV2(input)
	if err != nil {
		return nil, err
	}

	return output, nil
}

//KeyExists checks if a key exists
func (c *Client) KeyExists(key string) (bool, error) {
	return c.BucketKeyExists(c.Bucket, key)
}

//BucketKeyExists checks if an S3 bucket and key exists
func (c *Client) BucketKeyExists(bucket, key string) (bool, error) {

	req := &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}
	_, err := c.svc.HeadObject(req)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case "NotFound":
				return false, nil
			default:
				return false, err
			}
		}
	}
	return true, nil
}

//DownloadFile downloads a key to a file in a local directory
func (c *Client) DownloadFile(key string, dst string) error {

	// Get the object destination path
	parts := strings.Split(key, "/")
	file := parts[len(parts)-1]
	objDst := filepath.Join(dst, file)

	req := &s3.GetObjectInput{
		Bucket: aws.String(c.Bucket),
		Key:    aws.String(key),
	}

	resp, err := c.svc.GetObject(req)
	if err != nil {
		return err
	}

	// Create all the parent directories
	if err := os.MkdirAll(filepath.Dir(objDst), 0755); err != nil {
		return err
	}

	f, err := os.Create(objDst)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)

	return err
}

//UploadDirectory uploads a local directory to s3
func (c *Client) UploadDirectory(prefix string, dir string) error {

	fileList := []string{}
	filepath.Walk(dir, func(path string, f os.FileInfo, err error) error {
		isDir, err := isDirectory(path)
		if err != nil {
			return err
		}
		if isDir {
			return nil // Do nothing
		}
		fileList = append(fileList, path)
		return nil
	})

	for _, file := range fileList {
		err := c.UploadFile(prefix, dir, file)
		if err != nil {
			return err
		}
	}
	return nil
}

//UploadFile uploads a file to s3
func (c *Client) UploadFile(prefix string, dir string, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	var key string
	fileDirectory, _ := filepath.Abs(filePath)

	//remove base local directory
	fileDirectory = strings.Split(fileDirectory, dir)[1]
	key = path.Join(prefix, fileDirectory)

	//infer content-type from file extension (default to text)
	contentType := ""
	switch extension := filepath.Ext(filePath); extension {
	case ".txt":
		contentType = "text/plain"
	case ".csv":
		contentType = "text/csv"
	case ".tsv":
		contentType = "text/tsv"
	case ".html":
		contentType = "text/html"
	case ".json":
		contentType = "application/json"
	case ".xml": //why not? :)
		contentType = "application/xml"
	}

	// Upload the file to the s3 given bucket
	params := &s3.PutObjectInput{
		Bucket: aws.String(c.Bucket),
		Key:    aws.String(key),
		Body:   file,
	}
	if contentType != "" {
		params.ContentType = aws.String(contentType)
	}
	_, err = c.svc.PutObject(params)
	if err != nil {
		return err
	}
	return nil
}

func isDirectory(path string) (bool, error) {
	fd, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	switch mode := fd.Mode(); {
	case mode.IsDir():
		return true, nil
	case mode.IsRegular():
		return false, nil
	}
	return false, nil
}

//GetPresignedURL gets a presigned URL for the specified key
func (c *Client) GetPresignedURL(key string, expiration time.Duration) (string, error) {
	req, _ := c.svc.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(c.Bucket),
		Key:    aws.String(key),
	})
	urlStr, err := req.Presign(expiration)
	if err != nil {
		return "", err
	}
	return urlStr, nil
}
