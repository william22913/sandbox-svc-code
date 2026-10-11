package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"time"

	"github.com/nexsoft-git/nexcommon/constanta"
	srvCtx "github.com/nexsoft-git/nexcommon/context"
	"github.com/nexsoft-git/nexcommon/model"
	"github.com/nexsoft-git/nexlogger/log"

	"github.com/nexsoft-git/nexcommon/dto/out"
	"github.com/nexsoft-git/nexcommon/server/metrics"
)

func NewAPIConnector(
	metrics metrics.Metrics,
) APIConnector {
	return &apiConnector{
		metrics:      metrics,
		logEveryHit:  true,
		outgoingPath: "/v1/audit/outgoing",
	}
}

type apiConnector struct {
	metrics              metrics.Metrics
	logEveryHit          bool
	resourceID           string
	outgoingPath         string
	externalActivityPath string
	fixedToken           string
}

func (a *apiConnector) EnableExternalActivity(
	resourceID string,
	externalActivityHost string,
	externalActivityPort int,
	fixedToken string,
) {
	a.resourceID = resourceID
	a.externalActivityPath = fmt.Sprintf("http://%s:%d", externalActivityHost, externalActivityPort)
	if externalActivityPort == 0 {
		a.externalActivityPath = fmt.Sprintf("https://%s", externalActivityHost)
	}

	a.fixedToken = fixedToken
}

func (a *apiConnector) DisableApiConnectorLog() {
	a.logEveryHit = false
}

type HTTPClientDTO struct {
	Header        http.Header
	Unsuccessfull out.DefaultErrorResponse
}

func (h *HTTPClientDTO) SetHeader(header http.Header) {
	h.Header = header
}

func (h *HTTPClientDTO) GetHeader() http.Header {
	return h.Header
}

func (h *HTTPClientDTO) Unsuccessfully() interface{} {
	return &h.Unsuccessfull
}

type AbsHTTPClientDTO interface {
	GetHeader() http.Header
	SetHeader(http.Header)
	Unsuccessfully() interface{}
}

type hitAPILog struct {
	FullPath string            `json:"path"`
	Method   string            `json:"method"`
	Header   map[string]string `json:"header"`
	Error    error             `json:"err"`
	Status   int               `json:"status"`
}

func (a apiConnector) HitAPI(
	ctx context.Context,
	method string,
	host string,
	path string,
	header map[string]string,
	body interface{},
	result AbsHTTPClientDTO,
) (
	int,
	error,
) {
	var status int
	var err error
	now := time.Now()

	defer func() {
		a.metrics.GetDefaultMetric().APIConnectorHist.WithLabelValues(
			host,
			path,
			method,
			strconv.Itoa(status),
		).Observe(float64(time.Since(now).Seconds()))

		if a.logEveryHit {

			metaData := hitAPILog{
				FullPath: fmt.Sprintf("%s%s", host, path),
				Method:   method,
				Header:   header,
				Error:    err,
				Status:   status,
			}

			data, _ := json.Marshal(metaData)
			ctxMdl, _ := ctx.Value(constanta.ApplicationContextConstanta).(*srvCtx.ContextModel)
			log := log.Info().
				Str("meta_data", string(data))
			if ctxMdl != nil {
				log = log.LoggerModel(ctxMdl.ClientAccess.Logger)
			}
			log.Msg("Do Hit API Another Service")
		}
	}()

	url := fmt.Sprintf("%s%s", host, path)
	bodyByte, _ := json.Marshal(body)

	req, err := http.NewRequest(method, url, bytes.NewBuffer(bodyByte))
	if err != nil {
		return 0, err
	}

	req.Close = true

	a.checkContext(ctx, header)
	a.addHeader(req, header)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}

	result.SetHeader(resp.Header)
	status = resp.StatusCode

	defer resp.Body.Close()

	bodyResponse, _ := ioutil.ReadAll(resp.Body)

	if len(bodyResponse) > 0 {
		if status == 200 {
			err = json.Unmarshal(bodyResponse, result)
		} else {
			err = json.Unmarshal(bodyResponse, result.Unsuccessfully())
		}

		if err != nil {
			return status, err
		}

	}

	return resp.StatusCode, nil
}

func (a apiConnector) HitAPIMultipart(
	ctx context.Context,
	method string,
	host string,
	path string,
	header map[string]string,
	body model.MultipartRequest,
	result AbsHTTPClientDTO,
) (
	int,
	error,
) {
	var status int
	var err error
	now := time.Now()

	defer func() {
		a.metrics.GetDefaultMetric().APIConnectorHist.WithLabelValues(
			host,
			path,
			method,
			strconv.Itoa(status),
		).Observe(float64(time.Since(now).Seconds()))

		if a.logEveryHit {
			ctxMdl, _ := ctx.Value(constanta.ApplicationContextConstanta).(*srvCtx.ContextModel)

			log.Info().
				Interface("meta_data", hitAPILog{
					FullPath: fmt.Sprintf("%s%s", host, path),
					Method:   method,
					Header:   header,
					Error:    err,
					Status:   status,
				}).
				LoggerModel(ctxMdl.ClientAccess.Logger).
				Msg("Do Hit API Another Service")
		}
	}()

	buffer := &bytes.Buffer{}
	writer := multipart.NewWriter(buffer)

	for keys := range body {
		switch body[keys].(type) {
		case model.FileURLPath:
			data := body[keys].(model.FileURLPath)

			part, err := writer.CreateFormFile(keys, filepath.Base(data.FullPath))
			if err != nil {
				return 0, err
			}

			file, err := os.OpenFile(data.FullPath, os.O_RDONLY, os.ModeAppend)

			if err != nil {
				return 0, err
			}

			defer func() {
				_ = file.Close()
				if err == nil {
					if !data.KeepFile {
						_ = os.Remove(data.FullPath)
					}
				}
			}()

			_, err = io.Copy(part, file)

		default:
			if reflect.ValueOf(body[keys]).Kind() == reflect.Struct {
				bodyByte, _ := json.Marshal(body[keys])
				_ = writer.WriteField(keys, string(bodyByte))
			} else {
				_ = writer.WriteField(keys, fmt.Sprintf("%v", body[keys]))
			}
		}
	}

	_ = writer.Close()

	url := fmt.Sprintf("%s%s", host, path)
	req, err := http.NewRequest(method, url, buffer)

	if err != nil {
		return 0, err
	}

	req.Close = true

	header["Content-Type"] = writer.FormDataContentType()

	a.checkContext(ctx, header)
	a.addHeader(req, header)

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		return 0, err
	}

	result.SetHeader(resp.Header)
	status = resp.StatusCode

	defer resp.Body.Close()

	bodyResponse, _ := ioutil.ReadAll(resp.Body)

	if len(bodyResponse) > 0 {
		if status == 200 {
			err = json.Unmarshal(bodyResponse, result)
		} else {
			err = json.Unmarshal(bodyResponse, result.Unsuccessfully())
		}

		if err != nil {
			return 0, err
		}

	}

	return resp.StatusCode, nil
}

func (a apiConnector) addHeader(req *http.Request, header map[string]string) {
	for keys := range header {
		req.Header.Add(keys, header[keys])
	}
}

func (a apiConnector) checkContext(ctx context.Context, header map[string]string) {
	if header == nil {
		header = make(map[string]string)
	}

	requestID := ctx.Value(constanta.RequestIDConstanta)
	if header[constanta.RequestIDConstanta] == "" {
		header[constanta.RequestIDConstanta], _ = requestID.(string)
	}

	ipAddress := ctx.Value(constanta.IPAddressConstanta)
	if header[constanta.IPAddressConstanta] == "" {
		header[constanta.IPAddressConstanta], _ = ipAddress.(string)
	}

	source := ctx.Value(constanta.SourceConstanta)
	if header[constanta.SourceConstanta] == "" {
		header[constanta.SourceConstanta], _ = source.(string)
	}
}
