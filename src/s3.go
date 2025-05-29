package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type S3Service struct {
	client     *s3.Client
	uploader   *manager.Uploader
	downloader *manager.Downloader
	bucketName string
}

func NewS3Service(ctx context.Context, region, endpoint, accessKey, secretKey, bucketName string) (*S3Service, error) {
	cfgOptions := []func(*config.LoadOptions) error{
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
	}

	if endpoint != "" {
		cfgOptions = append(cfgOptions, config.WithBaseEndpoint(endpoint))
	}

	cfg, err := config.LoadDefaultConfig(ctx, cfgOptions...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	s3Client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		if endpoint != "" {
			o.UsePathStyle = true
		}

		// if it's still not working after 3 retries, give up. (like i did)
		o.RetryMaxAttempts = 3
		o.RetryMode = aws.RetryModeAdaptive
	})

	uploader := manager.NewUploader(s3Client, func(u *manager.Uploader) {
		u.PartSize = 10 * 1024 * 1024 // 10MB Parts
		u.Concurrency = 999999        // give me all the power! (i mean, threads)
	})

	downloader := manager.NewDownloader(s3Client, func(d *manager.Downloader) {
		d.PartSize = 10 * 1024 * 1024 // 10MB Parts
		d.Concurrency = 999999
	})

	return &S3Service{
		client:     s3Client,
		uploader:   uploader,
		downloader: downloader,
		bucketName: bucketName,
	}, nil
}

func (s *S3Service) UploadToS3(ctx context.Context, file io.Reader, filename string) error {

	_, err := s.uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(filename),
		Body:   file,
	})
	if err != nil {
		return fmt.Errorf("failed to upload file '%s' to S3 bucket '%s': %w", filename, s.bucketName, err)
	}

	return nil
}

func (s *S3Service) DownloadFromS3(ctx context.Context, filename string) (io.ReadCloser, error) {

	buffer := manager.NewWriteAtBuffer([]byte{})

	_, err := s.downloader.Download(ctx, buffer, &s3.GetObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(filename),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to download file '%s' from S3 bucket '%s': %w", filename, s.bucketName, err)
	}

	return io.NopCloser(bytes.NewReader(buffer.Bytes())), nil
}

// the difference is that the file isn't downloaded first, hopefully this incerases speed a bit
func (s *S3Service) StreamFromS3(ctx context.Context, filename string) (io.ReadCloser, error) {
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(filename),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to stream file '%s': %w", filename, err)
	}

	return out.Body, nil
}

func (s *S3Service) GetContentLength(ctx context.Context, filename string) (int64, error) {
	headOutput, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(filename),
	})
	if err != nil {
		return 0, fmt.Errorf("failed to get content length for file '%s' from S3 bucket '%s': %w", filename, s.bucketName, err)
	}
	return *headOutput.ContentLength, nil
}

func (s *S3Service) DoesFileExistInS3(ctx context.Context, filename string) (bool, error) {

	_, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(filename),
	})

	if err != nil {

		var nsk *types.NoSuchKey
		var nf *types.NotFound

		if errors.As(err, &nsk) || errors.As(err, &nf) {

			return false, nil
		}

		return false, fmt.Errorf("failed to check existence of file '%s' in S3 bucket '%s': %w", filename, s.bucketName, err)
	}

	return true, nil
}
