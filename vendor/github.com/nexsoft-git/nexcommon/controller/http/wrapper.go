package http

import (
	"database/sql"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"runtime/debug"
	"time"

	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/nexsoft-git/nexcommon/constanta"
	internalCtx "github.com/nexsoft-git/nexcommon/context"
	"github.com/nexsoft-git/nexcommon/controller"
	"github.com/nexsoft-git/nexcommon/docs/swagger"
	"github.com/nexsoft-git/nexcommon/dto/in"
	"github.com/nexsoft-git/nexcommon/dto/out"
	errors "github.com/nexsoft-git/nexcommon/error"
	"github.com/nexsoft-git/nexcommon/idempotency"
	"github.com/nexsoft-git/nexcommon/model"
	srv "github.com/nexsoft-git/nexcommon/server"
	"github.com/nexsoft-git/nexcommon/util/text"
	"github.com/nexsoft-git/nexcommon/util/validator/multipart_validator"
	"github.com/nexsoft-git/nexlogger/log"
)

type WrapperServiceResult struct {
	f  func(http.ResponseWriter, *http.Request)
	cs swagger.SwaggerControllerService
}

// writePanicRecoveryResponse writes a minimal, hardcoded 500 in the
// service's normal {success, header, payload} envelope shape after a
// panic has already been recovered and logged. Before this existed,
// the recover() branches below logged the panic but wrote nothing to
// rw at all — the handler then returned with no response ever sent,
// so the client held the connection open until the server's own
// ReadTimeout/WriteTimeout (or indefinitely, on a service that never
// set one), and whatever status eventually resulted was never an
// intentional 500 in this service's own response shape.
//
// Deliberately NOT the normal ctx/g.formator-based response path used
// everywhere else in this file: the panic may have happened before
// ctx was even constructed, or before the request made it far enough
// for g.formator to be safely usable, and calling into that path from
// inside an already-recovering deferred func risks a second panic
// that Go will NOT recover from (a panic inside a deferred function
// that already called recover() is not itself recoverable by that
// same defer). Every value written here is a static literal for
// exactly that reason.
//
// Known, accepted gap: if the panic happens AFTER this request's
// response headers were already written (e.g. mid-body-write), this
// call becomes a harmless "superfluous WriteHeader" no-op for the
// status, but the body write below still appends after whatever
// bytes already went out, which can produce a malformed response body
// in that specific case. That's a pre-existing limitation of
// recovering after headers are sent, not something this change
// introduces — the alternative (silence, the prior behavior) is
// strictly worse for the overwhelmingly common case: a panic before
// any bytes were written at all.
func writePanicRecoveryResponse(rw http.ResponseWriter) {
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusInternalServerError)
	_, _ = rw.Write([]byte(`{"success":false,"payload":{"status":500,"code":"E-5-CMD-SRV-001","message":"Internal Server Error"}}`))
}

func (g HTTPController) WrapService(
	param *wrapServiceParam,
) WrapperServiceResult {

	dto := param.service.GetDTO()

	if param.multipart || param.isURLEncoded {
		dto = param.service.GetMultipartDTO()
	}

	service := g.swaggerGenerator.ReadDTO(
		param.multipart,
		param.isURLEncoded,
		dto,
		param.menu,
	)

	var swaggerParam []interface{}

	if param.cv.Name != controller.WHITELIST_TOKEN_VALIDATOR {
		swaggerParam = append(
			swaggerParam,
			swagger.SwaggerParametersWithoutReferences{
				In:   "header",
				Name: constanta.NexAuthResultKey,
				Schema: swagger.SwaggerSchema{
					Type: "string",
				},
			},
		)

		service.Security = append(service.Security, swagger.GenerateServiceSecurity(param.cv.Name))
	}

	if !param.readBody {
		service.RequestBody = nil
	} else {
		if service.RequestBody != nil {
			service.RequestBody = map[string]interface{}{"content": service.RequestBody}
		}
	}

	if param.isIdempotency {
		swaggerParam = append(
			swaggerParam,
			swagger.SwaggerParametersWithoutReferences{
				In:          "header",
				Name:        constanta.IdempotencyHeaderKey,
				Description: fmt.Sprintf("Set Your Idempotency Key (We Hold the response for %d Minute(s))", idempotency.IDEMPOTENCY_CLEANUP_DURATION),
				Required:    true,
				Schema: swagger.SwaggerSchema{
					Type:    "string",
					Example: text.GetUUID(),
				},
			},
		)
	}

	//Path Request
	for i := 0; i < len(param.pathParams); i++ {
		name := param.pathParams[i]

		dataType, min := resolvePathParamSwaggerType(name, param.pathParamTypes)

		example := ""
		if min != 0 {
			example = fmt.Sprintf("min = %d", min)
		}

		swaggerParam = append(swaggerParam,
			swagger.SwaggerParametersWithoutReferences{
				In:       "path",
				Name:     param.pathParams[i],
				Required: true,
				Schema: swagger.SwaggerSchema{
					Type:    dataType,
					Example: example,
				},
			},
		)
	}

	service.Description = param.description
	service.Tags = param.tags
	service.Parameters = swaggerParam

	service.Responses = make(map[int]interface{})
	service.Responses[400] = swagger.SwaggerReferences{
		Ref: "#/components/responses/Error",
	}

	if param.output != nil {
		resp := swagger.ReadResponse(param.dtoParam.fileResult, param.dtoParam.description, param.output)
		resp.Description = param.dtoParam.description
		service.Responses[200] = resp
	}

	return WrapperServiceResult{
		f: func(
			rw http.ResponseWriter,
			r *http.Request,
		) {
			var ctx *internalCtx.ContextModel
			var err error
			var payload interface{}
			var header map[string]string
			var urlParam model.URLParam
			var idempotencyKey string
			var processing *idempotency.ServiceProcess
			var serviceResult model.IdempotencyServiceProcessResult
			timeNow := time.Now()

			defer func() {
				if rcr := recover(); rcr != nil {
					log.Error().
						Caller().
						Interface("panic", rcr).
						Str("stack", string(debug.Stack())).
						Msg("Panic recovered")
					writePanicRecoveryResponse(rw)
				} else {
					if param.isIdempotency {
						payload, header, err = g.doFinishIdempotencyProcess(
							ctx,
							payload,
							header,
							processing,
							idempotencyKey,
							serviceResult,
							err,
						)
					}

					g.response(
						ctx,
						r,
						rw,
						param.isSendSignature,
						timeNow,
						header,
						payload,
						err,
						param.isRecordActivityActive,
						param.recordActivity,
						urlParam,
					)
				}
			}()

			ctx = g.getContext(
				r,
				param.isCheckSignature,
				param.isSendSignature,
			)

			err = g.preStart(
				r,
				param,
			)

			if err != nil {
				err = errors.ErrReadBody
				return
			}

			ctx.ClientAccess.Path = r.URL.Path
			// Validate Token
			_, headers := g.Converter(r)

			ctx.ClientAccess.Timestamp, err = time.Parse(constanta.DefaultDtoOutTimeFormat, headers[constanta.ClientRequestTimestamp])
			if err != nil {
				ctx.ClientAccess.Timestamp = time.Now()
			}

			err = param.cv.Func(ctx.ToContext(), headers)
			if err != nil {
				return
			}

			c, valid := r.Context().Value(
				constanta.ApplicationContextConstanta,
			).(*internalCtx.ContextModel)

			if valid {
				ctx = c
			}

			err = g.PermissionAndScopeChecker(
				ctx,
				param.permission,
				param.apiScope,
				param.dataScope,
			)

			if err != nil {
				return
			}

			var dto interface{}

			if param.multipart || param.isURLEncoded {

				if param.isURLEncoded {
					err = r.ParseForm()

					if err != nil {
						return
					}
				}

				dto, err = g.multipartValidator.ValidateMultipart(
					multipart_validator.NewValidateMultipartParam(
						ctx,
						r,
						param.service.GetMultipartDTO(),
					).Menu(
						param.menu,
					).FunctionValidator(
						param.functionValidator,
					),
				)
			} else {
				dto, err = g.readBody(
					ctx,
					param,
					r,
				)
			}

			if err != nil {
				return
			}

			urlParam, ctx.ClientAccess.Headers = g.getURLParam(
				r,
				param.pathParams,
				param.headers,
			)

			if param.isIdempotency {
				idempotencyKey, processing, serviceResult, err = g.doStartIdempotencyProcess(ctx, r)

				if err != nil || serviceResult.Output != nil {
					return
				}
			}

			header, payload, err = param.serve(
				ctx,
				urlParam,
				dto,
			)

		},
		cs: service,
	}
}

func (g HTTPController) doStartIdempotencyProcess(
	ctx *internalCtx.ContextModel,
	r *http.Request,
) (
	idempotencyKey string,
	processing *idempotency.ServiceProcess,
	serviceResult model.IdempotencyServiceProcessResult,
	err error,
) {
	idempotencyKey = r.Header.Get(constanta.IdempotencyHeaderKey)

	if idempotencyKey == "" {
		err = errors.ErrEmptyField.ContextModel(ctx).Param("IDEMPOTENCY_KEY")
		return
	}

	ctx.ClientAccess.IdempotencyKey = idempotencyKey
	processing, err = g.GetIdempotencyProcessing().AddProcessingService(ctx.ClientAccess.Path, idempotencyKey)

	if err != nil {
		return
	}

	serviceResult = processing.GetServiceProcessResult()

	return
}

func (g HTTPController) doFinishIdempotencyProcess(
	ctx *internalCtx.ContextModel,
	payload interface{},
	header map[string]string,
	processing *idempotency.ServiceProcess,
	idempotencyKey string,
	serviceResult model.IdempotencyServiceProcessResult,
	err error,
) (
	interface{},
	map[string]string,
	error,
) {
	defer func() {
		if processing != nil && processing.Tx != nil {
			err = processing.Tx.Commit()
		}
	}()

	if err == nil {
		if serviceResult.Output == nil {
			idempotencyResult := model.IdempotencyServiceProcessResult{
				Output: payload,
				Header: header,
				Type:   reflect.TypeOf(payload).String(),
			}

			if processing.Func != nil {
				err = processing.Func(
					model.IdempotencyModel{
						IdempotencyKey: sql.NullString{String: idempotency.ToIdempotencyKey(ctx.ClientAccess.Path, idempotencyKey)},
						Result:         idempotencyResult,
					},
				)

				if err != nil {
					return nil, nil, err
				}

			} else {
				processing.StopProcess(idempotencyResult)
			}
		} else {
			header = serviceResult.Header

			switch serviceResult.Type {
			case reflect.TypeOf(model.RedirectURL{}).String():
				byteData, _ := json.Marshal(serviceResult.Output)
				temp := model.RedirectURL{}
				_ = json.Unmarshal(byteData, &temp)
				payload = temp
			case reflect.TypeOf(model.FileURLPath{}).String():
				byteData, _ := json.Marshal(serviceResult.Output)
				temp := model.FileURLPath{}
				_ = json.Unmarshal(byteData, &temp)
				payload = temp
			default:
				payload = serviceResult.Output
			}
			processing.Close()
		}
	}

	return payload, header, err
}

func (g HTTPController) WrapServiceListData(
	param *wrapGetListServiceParam,
) WrapperServiceResult {
	return g.wrapServiceListData(
		param,
		false,
	)
}

func (g HTTPController) WrapServiceCountData(
	param *wrapGetListServiceParam,
) WrapperServiceResult {
	return g.wrapServiceListData(
		param,
		true,
	)
}

func (g HTTPController) wrapServiceListData(
	param *wrapGetListServiceParam,
	count bool,
) WrapperServiceResult {

	service := g.swaggerGenerator.ReadDTO(false, false, &struct{}{}, "")
	if service.RequestBody != nil {
		service.RequestBody = map[string]interface{}{"content": service.RequestBody}
	}

	var swaggerParam []interface{}

	//Authorization
	if param.cv.Name != controller.WHITELIST_TOKEN_VALIDATOR {
		swaggerParam = append(
			swaggerParam,
			swagger.SwaggerParametersWithoutReferences{
				In:   "header",
				Name: constanta.NexAuthResultKey,
				Schema: swagger.SwaggerSchema{
					Type: "string",
				},
			},
		)

		service.Security = append(service.Security, swagger.GenerateServiceSecurity(param.cv.Name))
	}

	if param.isIdempotency {
		swaggerParam = append(
			swaggerParam,
			swagger.SwaggerParametersWithoutReferences{
				In:          "header",
				Name:        constanta.IdempotencyHeaderKey,
				Description: fmt.Sprintf("Set Your Idempotency Key (We Hold the response for %d Minute(s))", idempotency.IDEMPOTENCY_CLEANUP_DURATION),
				Required:    true,
				Schema: swagger.SwaggerSchema{
					Type:    "string",
					Example: text.GetUUID(),
				},
			},
		)
	}

	service.Responses = make(map[int]interface{})
	service.Responses[400] = swagger.SwaggerReferences{
		Ref: "#/components/responses/Error",
	}

	if count {
		service.Responses[200] = swagger.SwaggerReferences{
			Ref: "#/components/responses/CountData",
		}
	} else {
		if param.output != nil {
			resp := swagger.ReadResponse(false, param.dtoParam.description, param.output)
			resp.Description = param.dtoParam.description
			service.Responses[200] = resp
		}
	}

	//Path Request
	for i := 0; i < len(param.pathParams); i++ {
		name := param.pathParams[i]

		dataType, min := resolvePathParamSwaggerType(name, param.pathParamTypes)

		example := ""
		if min != 0 {
			example = fmt.Sprintf("min = %d", min)
		}

		swaggerParam = append(swaggerParam,
			swagger.SwaggerParametersWithoutReferences{
				In:       "path",
				Name:     param.pathParams[i],
				Required: true,
				Schema: swagger.SwaggerSchema{
					Type:    dataType,
					Example: example,
				},
			},
		)
	}

	//GET LIST PARAM
	// Part 1 item 12 (second half, swagger-branch-only): page, limit,
	// filter, order, and connector were all documented as Required:
	// true here, despite every one of them being genuinely optional —
	// page now defaults to 1 (ValidateInputPageLimitAndOrderBy), limit
	// defaults via checkLimit, order defaults to
	// param.defaultOrder/GetListValidOrderBy()[0], connector defaults
	// to "AND" below, and filter has no default because omitting it
	// just means "no filtering" (not an error). A generated client
	// that took this document at face value would send all five on
	// every request for no reason, and would have no way to know
	// omitting any of them is actually fine.
	if !count {
		description := fmt.Sprintf("min = %d", 1)
		swaggerParam = append(
			swaggerParam,
			swagger.SwaggerParametersWithoutReferences{
				In:          "query",
				Name:        "page",
				Description: description,
				Required:    false,
				Schema: swagger.SwaggerSchema{
					Type:    "integer",
					Example: "1",
				},
			},
		)

		swaggerParam = append(
			swaggerParam,
			swagger.SwaggerParametersWithoutReferences{
				In:       "query",
				Name:     "limit",
				Required: false,
				Schema: swagger.SwaggerSchema{
					Type:    "integer",
					Example: fmt.Sprintf("%d", param.service.GetListValidLimit()[0]),
					Enum:    param.service.GetListValidLimit(),
				},
			},
		)

		swaggerParam = append(
			swaggerParam,
			swagger.SwaggerParametersWithoutReferences{
				In:       "query",
				Name:     "filter",
				Required: false,
				Schema: swagger.SwaggerSchema{
					Type:    "string",
					Example: "name||description&&title in rizky-test-test2, price&&qty in 10-30-50, qty eq 100, created_at rng 2020-01-01T00:00:00Z>>2021-01-01T00:00:00Z",
				},
			},
		)

		swaggerParam = append(
			swaggerParam,
			swagger.SwaggerParametersWithoutReferences{
				In:          "query",
				Name:        "order",
				Required:    false,
				Description: "Order List Result, if only fill by field name, will order on ASC mode",
				Schema: swagger.SwaggerSchema{
					Type:    "string",
					Example: "field_1 <ASC>/<DESC>",
				},
			},
		)

		swaggerParam = append(
			swaggerParam,
			swagger.SwaggerParametersWithoutReferences{
				In:          "query",
				Name:        "connector",
				Required:    false,
				Description: "Connector for every filter, Default = AND",
				Schema: swagger.SwaggerSchema{
					Type:    "string",
					Example: "<AND>/<OR>",
				},
			},
		)
	}

	service.Description = param.description
	service.Tags = param.tags
	service.Parameters = swaggerParam

	return WrapperServiceResult{
		f: func(rw http.ResponseWriter, r *http.Request) {
			var ctx *internalCtx.ContextModel
			var err error
			var payload interface{}
			var header map[string]string
			var urlParam model.URLParam
			var idempotencyKey string
			var processing *idempotency.ServiceProcess
			var serviceResult model.IdempotencyServiceProcessResult
			timeNow := time.Now()

			defer func() {
				if rcr := recover(); rcr != nil {
					log.Error().Caller().
						Interface("panic", rcr).
						Str("stack", string(debug.Stack())).
						Msg("Panic recovered")
					writePanicRecoveryResponse(rw)
				} else {
					if param.isIdempotency {
						payload, header, err = g.doFinishIdempotencyProcess(
							ctx,
							payload,
							header,
							processing,
							idempotencyKey,
							serviceResult,
							err,
						)
					}

					g.response(
						ctx,
						r,
						rw,
						param.isSendSignature,
						timeNow,
						header,
						payload,
						err,
						param.isRecordActivityActive,
						param.recordActivity,
						urlParam,
					)
				}
			}()

			ctx = g.getContext(
				r,
				param.isCheckSignature,
				param.isSendSignature,
			)

			r.Header.Add(
				constanta.NexAuthHeaderScopeConstanta,
				param.apiScope,
			)

			ctx.ClientAccess.Path = r.URL.Path
			// Validate Token
			_, headers := g.Converter(r)

			ctx.ClientAccess.Timestamp, err = time.Parse(constanta.DefaultDtoOutTimeFormat, headers[constanta.ClientRequestTimestamp])
			if err != nil {
				ctx.ClientAccess.Timestamp = time.Now()
			}

			err = param.cv.Func(ctx.ToContext(), headers)
			if err != nil {
				return
			}

			err = g.PermissionAndScopeChecker(
				ctx,
				param.permission,
				param.apiScope,
				param.dataScope,
			)

			if err != nil {
				return
			}

			dto := in.GetListRequest{}

			if r.Method == http.MethodGet {
				urlParam, ctx.ClientAccess.Headers = g.getURLParam(r, param.pathParams, param.headers)
				dto.Page, _ = strconv.Atoi(urlParam.Query["page"])
				dto.Limit, _ = strconv.Atoi(urlParam.Query["limit"])
				dto.Filter = urlParam.Query["filter"]
				dto.Order = strings.Trim(urlParam.Query["order"], " ")
				dto.Connector = strings.Trim(urlParam.Query["connector"], " ")
				dto.OrderIDs = strings.Trim(urlParam.Query["order_ids"], " ")
				dto.OrderIDOperator = strings.Trim(urlParam.Query["order_id_opt"], " ")
			} else {

				var stringBody string
				stringBody, ctx.ClientAccess.Logger.ByteIn, err = srv.ReadBodyWithLimit(rw, r, g.maxRequestBodyBytes)

				if err != nil {
					if stderrors.Is(err, srv.ErrBodyTooLarge) {
						err = errors.ErrRequestBodyTooLarge
					}
					return
				}

				if param.recordActivity.saveRequest {
					param.recordActivity.body = stringBody
					param.recordActivity.header = ctx.ClientAccess.Headers
				}

				err = json.Unmarshal([]byte(stringBody), &dto)

				if err != nil {
					err = errors.ErrMarshalingBody
					return
				}
			}

			if dto.Connector == "" {
				dto.Connector = "AND"
			}

			if dto.Order == "" {
				dto.Order = param.defaultOrder
			}

			var searchParam []model.SearchParam

			if count {
				searchParam, err = g.listDataValidator.ValidateGetCountData(
					ctx,
					&dto,
					param.service,
				)
			} else {
				dto.EnableTokenChecker = param.isEnableNextTokenChecker
				searchParam, err = g.listDataValidator.ValidateGetListData(
					ctx,
					&dto,
					param.service,
				)
			}

			if err != nil {
				return
			}

			if param.isIdempotency {
				idempotencyKey, processing, serviceResult, err = g.doStartIdempotencyProcess(ctx, r)

				if err != nil || serviceResult.Output != nil {
					return
				}
			}

			header, payload, err = param.serve(
				ctx,
				searchParam,
				dto,
			)

		},
		cs: service,
	}
}

func (g HTTPController) response(
	ctx *internalCtx.ContextModel,
	r *http.Request,
	rw http.ResponseWriter,
	sendSignature bool,
	timeNow time.Time,
	header map[string]string,
	payload interface{},
	err error,
	isRecordActivityActive bool,
	recordActivity recordActivity,
	path_param model.URLParam,
) {
	var errs out.DefaultErrorResponse
	var usedErr = err
	statusCode := 200

	defer func() {
		usedPath, _ := mux.CurrentRoute(r).GetPathTemplate()
		if usedPath == "" {
			usedPath = r.URL.Path
		}

		if g.logEveryHit {
			ctx.ClientAccess.Logger.ProcessingTime = time.Since(timeNow).Microseconds()
			if !g.basicValidator.ValidateStringContainInStringArray(
				g.skipEndpointsLog,
				r.URL.Path,
			) {

				msg := "Api Called"
				if errs.Payload.Message != "" {
					msg = errs.Payload.Message
					ctx.ClientAccess.Logger.Code = errs.Payload.Code
				}

				logEvent := log.Info().
					LoggerModel(ctx.ClientAccess.Logger).
					Str("action", "middleware.api.call").
					Str("url", usedPath).
					Str("method", r.Method)

				if usedErr != nil {
					logEvent = logEvent.Err(usedErr)
				}

				logEvent.Msg(msg)
			}
		}

		if g.activityUtil != nil &&
			isRecordActivityActive &&
			(usedErr == nil || (g.activityUtil.IsLogAllActivity() && statusCode != 500)) {

			recordActivity.activity["path"] = usedPath
			recordActivity.activity["method"] = r.Method
			recordActivity.activity["app"] = ctx.ClientAccess.Logger.Application

			if path_param.Path != nil && len(path_param.Path) > 0 {
				recordActivity.activity["path_param"] = path_param.Path
			}

			activity := model.ActivityDTO{
				MenuCode:          recordActivity.menuCode,
				TimeActivity:      timeNow,
				IPMac:             ctx.ClientAccess.Logger.IP,
				IsMenuAdmin:       recordActivity.isMenuAdmin,
				Activity:          recordActivity.activity,
				CreatedBy:         ctx.Limitation.ServiceUserID,
				CreatedClient:     ctx.ClientAccess.Logger.ClientID,
				CreatedAt:         time.Now(),
				RequestTimestamp:  ctx.ClientAccess.Timestamp,
				ResponseTimestamp: time.Now(),
				ResponseStatus:    statusCode,
			}

			if recordActivity.saveRequest {
				activity.Body = recordActivity.body
				activity.Header = text.StructToJSON(recordActivity.header)
			}

			if recordActivity.externalActivity {
				g.externalAcivityUtil.PushActivity(activity)
			}

			if recordActivity.localActivity {
				g.activityUtil.PushActivity(activity)
			}

		}
	}()

	rw.Header().Set(constanta.RequestIDConstanta, ctx.ClientAccess.Logger.RequestID)
	forceCode, valid := header[constanta.ForceHeaderCodeResponse]

	if valid {

		statusCodeForced, _ := strconv.Atoi(forceCode)

		if statusCodeForced != 0 {
			statusCode = statusCodeForced
		}

		delete(header, constanta.ForceHeaderCodeResponse)

	}

	for key := range header {
		headersList := rw.Header().Get("Access-Control-Allow-Headers") + ", " + strings.ToLower(key)
		rw.Header().Set("Access-Control-Allow-Headers", headersList)
		rw.Header().Set("Access-Control-Expose-Headers", headersList)
		rw.Header().Add(key, header[key])
	}

	var data []byte
	var length int
	if err != nil {
		errs = g.formator.ReformatErrorMessage(
			*errors.NewErrorMessageParam(
				err,
			).WithContext(
				ctx,
			),
		)

		statusCode = errs.Payload.Status
		payload = errs

	} else {
		switch data := payload.(type) {
		case model.RedirectURL:
			http.Redirect(rw, r, data.NewURLPath, http.StatusFound)
			return
		case model.FileURLPath:
			var length int64
			var file *os.File

			defer func() {
				_ = file.Close()

				if !data.KeepFile {
					_ = os.Remove(data.FullPath)
				}

			}()

			file, err = os.OpenFile(data.FullPath, os.O_RDONLY, os.ModeAppend)

			if err != nil {
				errs = g.formator.ReformatErrorMessage(
					*errors.NewErrorMessageParam(
						err,
					).WithContext(
						ctx,
					),
				)

				statusCode = errs.Payload.Status
				payload = errs
				break
			}

			stat, _ := file.Stat()

			fileName := strings.Split(file.Name(), "/")
			rw.Header().Set("Content-Type", "application/octet-stream")
			rw.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", strings.Replace(fileName[len(fileName)-1], " ", "_", -1)))

			headersList := rw.Header().Get("Access-Control-Allow-Headers") + ", content-disposition"
			rw.Header().Set("Access-Control-Allow-Headers", headersList)
			rw.Header().Set("Access-Control-Expose-Headers", headersList)
			rw.Header().Set("Content-Length", fmt.Sprintf("%d", stat.Size()))

			length, err = io.Copy(rw, file)

			if err != nil {
				errs = g.formator.ReformatErrorMessage(
					*errors.NewErrorMessageParam(
						err,
					).WithContext(
						ctx,
					),
				)

				statusCode = errs.Payload.Status
				payload = errs
				break
			}

			ctx.ClientAccess.Logger.ByteOut = int(length)
			return

		default:
			payload = dtoConverter.GetFunction()(ctx, payload)
		}
	}

	rw.Header().Set("Content-Type", "application/json")
	ctx.ClientAccess.Logger.Status = statusCode
	rw.WriteHeader(statusCode)

	if data == nil {
		data, err = json.Marshal(payload)
		if err != nil {
			log.Error().
				LoggerModel(ctx.ClientAccess.Logger).
				Err(err).
				Caller().
				Msg("Error on Marshaling Message")
			return
		}

		if statusCode == http.StatusOK && sendSignature {
			timestamp := time.Now().Format(constanta.DefaultTimeFormat)
			digest := text.GenerateMessageDigest(string(data))
			signature := text.GenerateSignature(
				r.Method,
				r.URL.Path,
				ctx.ClientAccess.Logger.AccessToken,
				digest,
				timestamp,
				ctx.AuthAccessTokenModel.SignatureKey,
			)

			rw.Header().Set(constanta.TimestampSignatureHeaderNameConstanta, timestamp)
			rw.Header().Set(constanta.SignatureHeaderNameConstanta, signature)
		}

		length, err = rw.Write(data)
		if err != nil {
			log.Error().
				LoggerModel(ctx.ClientAccess.Logger).
				Err(err).
				Caller().
				Msg("Error on Writing Message")
			return
		}
	}

	ctx.ClientAccess.Logger.ByteOut = length

}
