package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
)

var uploadCmd = &Command{
	Name:        "upload",
	Summary:     "upload a file to the app",
	Synopsis:    "FILE_PATH",
	Description: "Uploads a local file using multipart/form-data. The response contains a fileId that can be stored in a record file field.",
	Arguments:   []string{"FILE_PATH: path of the local file to upload."},
	Examples:    []string{"upload ./report.pdf"},
	Endpoints:   []string{"POST /files"},
	Handler:     (*App).upload,
}

var downloadCmd = &Command{
	Name:        "download",
	Summary:     "download an app file to disk",
	Synopsis:    "FILE_ID OUTPUT_PATH | ENTITY RECORD_ID FILE_ID OUTPUT_PATH",
	Description: "Downloads an app file or a record-associated file and writes its bytes to OUTPUT_PATH. Use - as OUTPUT_PATH to write to standard output.",
	Arguments:   []string{"FILE_ID: uploaded file identifier.", "OUTPUT_PATH: destination path, or - for standard output.", "ENTITY RECORD_ID: optional record context to download a record file."},
	Examples:    []string{"download 588f9f1eb45fc9000bc1c010 ./report.pdf", "download companies r1 file-123 ./attachment.pdf"},
	Endpoints:   []string{"GET /files/{id}", "GET /data/{entityName}/{recordId}/files/{fileId}"},
	Handler:     (*App).download,
}

func (a *App) upload(ctx context.Context, args []string) error {
	if len(args) != 1 {
		return errors.New("usage: upload FILE_PATH")
	}
	token, err := a.requiredToken()
	if err != nil {
		return err
	}
	file, err := os.Open(args[0])
	if err != nil {
		return fmt.Errorf("open upload file: %w", err)
	}
	defer file.Close()

	reader, writer := io.Pipe()
	multipartWriter := multipart.NewWriter(writer)
	contentType := multipartWriter.FormDataContentType()
	go func() {
		part, err := multipartWriter.CreateFormFile("file", filepath.Base(args[0]))
		if err == nil {
			_, err = io.Copy(part, file)
		}
		if closeErr := multipartWriter.Close(); err == nil {
			err = closeErr
		}
		_ = writer.CloseWithError(err)
	}()

	response, err := a.requestWithOptions(ctx, http.MethodPost, "/files", nil, reader, contentType, "application/json", token)
	if err != nil {
		return err
	}
	return a.writeResponse(response)
}

func (a *App) download(ctx context.Context, args []string) error {
	var path, outputPath string
	switch len(args) {
	case 2:
		path = "/files/" + url.PathEscape(args[0])
		outputPath = args[1]
	case 4:
		path = "/data/" + url.PathEscape(args[0]) + "/" + url.PathEscape(args[1]) + "/files/" + url.PathEscape(args[2])
		outputPath = args[3]
	default:
		return errors.New("usage: download FILE_ID OUTPUT_PATH | download ENTITY RECORD_ID FILE_ID OUTPUT_PATH")
	}
	token, err := a.requiredToken()
	if err != nil {
		return err
	}
	response, err := a.requestWithOptions(ctx, http.MethodGet, path, nil, nil, "", "*/*", token)
	if err != nil {
		return err
	}
	if outputPath == "-" {
		_, err := a.output.Write(response)
		return err
	}
	destination, err := os.OpenFile(outputPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("create download file: %w", err)
	}
	if _, err := destination.Write(response); err != nil {
		_ = destination.Close()
		return fmt.Errorf("write download file: %w", err)
	}
	if err := destination.Close(); err != nil {
		return fmt.Errorf("close download file: %w", err)
	}
	return nil
}
