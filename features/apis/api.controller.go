package apis

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	models "schoolify_be/models"
	repoPostgres "schoolify_be/repositories/postgresql"
	utilities "schoolify_be/utilities"
	errUtility "schoolify_be/utilities/errors"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

var validate *validator.Validate

func init() {
	validate = validator.New()
}

type apiServiceInterface interface {
	GetAllSrvc(ctx context.Context, param models.GetAllApiRequest) ([]models.ApiResponse, int, errUtility.CustomError)
	CreateApiSrvc(ctx context.Context, param models.CreateApiRequest) errUtility.CustomError
	UpdateApiSrvc(ctx context.Context, apiID uuid.UUID, param models.UpdateApiRequest) errUtility.CustomError
}

type apiController struct {
	service apiServiceInterface
}

func (c apiController) createCtrl(w http.ResponseWriter, r *http.Request) {
	reqID := utilities.GetReqID(r.Context())
	path := r.URL.Path

	var reqBody models.CreateApiRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		utilities.Log(reqID, utilities.ERROR, path, "apiController.createCtrl", nil, "json decode failed", err)
		utilities.WriteJSON(w, http.StatusBadRequest, map[string]interface{}{
			"message": "Validation failed",
			"errors":  map[string][]string{"data": {"Invalid data"}},
		})
		return
	}

	utilities.Log(reqID, utilities.DEBUG, path, "apiController.createCtrl", reqBody, "incoming request", nil)
	if err := validate.Struct(reqBody); err != nil {
		errMap := errUtility.ParseError(err)
		utilities.Log(reqID, utilities.WARN, path, "apiController.createCtrl", reqBody, "validation failed", errMap)
		utilities.WriteJSON(w, http.StatusUnprocessableEntity, map[string]interface{}{
			"message": "Validation failed",
			"errors":  errMap,
		})
		return
	}

	if errObj := c.service.CreateApiSrvc(r.Context(), reqBody); errObj.Code > 0 {
		errObj.Compile()
		statusCode := int(errObj.Code)
		logLevel := utilities.ERROR
		if statusCode < http.StatusInternalServerError {
			logLevel = utilities.WARN
		}
		utilities.Log(reqID, logLevel, path, "apiController.createCtrl", reqBody, "service execution failed", errObj)

		if statusCode == http.StatusInternalServerError {
			utilities.WriteJSON(w, statusCode, map[string]interface{}{
				"message": "Failed to create API",
				"errors":  "Internal Server Error",
			})
			return
		}

		errKey := errObj.Property
		if errKey == "" {
			errKey = "error"
		}
		utilities.WriteJSON(w, statusCode, map[string]interface{}{
			"message": "Failed to create API",
			"errors":  map[string]string{errKey: errObj.MessageToSend},
		})
		return
	}

	utilities.Log(reqID, utilities.DEBUG, path, "apiController.createCtrl", nil, "success", nil)
	utilities.WriteJSON(w, http.StatusCreated, map[string]string{
		"message": "API created successfully",
	})
}

func (c apiController) updateCtrl(w http.ResponseWriter, r *http.Request) {
	reqID := utilities.GetReqID(r.Context())
	path := r.URL.Path
	apiIDStr := chi.URLParam(r, "apiID")
	if apiIDStr == "" {
		utilities.WriteJSON(w, http.StatusBadRequest, map[string]interface{}{
			"message": "Validation failed",
			"errors":  map[string][]string{"apiID": {"apiID path parameter is required"}},
		})
		return
	}

	apiID, err := uuid.Parse(apiIDStr)
	if err != nil {
		utilities.Log(reqID, utilities.WARN, path, "apiController.updateCtrl", map[string]string{"apiID": apiIDStr}, "invalid apiID", err)
		utilities.WriteJSON(w, http.StatusBadRequest, map[string]interface{}{
			"message": "Validation failed",
			"errors":  map[string][]string{"apiID": {"Invalid apiID"}},
		})
		return
	}

	var reqBody models.UpdateApiRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		utilities.Log(reqID, utilities.ERROR, path, "apiController.updateCtrl", nil, "json decode failed", err)
		utilities.WriteJSON(w, http.StatusBadRequest, map[string]interface{}{
			"message": "Validation failed",
			"errors":  map[string][]string{"data": {"Invalid data"}},
		})
		return
	}

	utilities.Log(reqID, utilities.DEBUG, path, "apiController.updateCtrl", reqBody, "incoming request", nil)
	if err := validate.Struct(reqBody); err != nil {
		errMap := errUtility.ParseError(err)
		utilities.Log(reqID, utilities.WARN, path, "apiController.updateCtrl", reqBody, "validation failed", errMap)
		utilities.WriteJSON(w, http.StatusBadRequest, map[string]interface{}{
			"message": "Validation failed",
			"errors":  errMap,
		})
		return
	}

	if errObj := c.service.UpdateApiSrvc(r.Context(), apiID, reqBody); errObj.Code > 0 {
		errObj.Compile()
		statusCode := int(errObj.Code)
		logLevel := utilities.ERROR
		if statusCode < http.StatusInternalServerError {
			logLevel = utilities.WARN
		}
		utilities.Log(reqID, logLevel, path, "apiController.updateCtrl", reqBody, "service execution failed", errObj)

		if statusCode == http.StatusInternalServerError {
			utilities.WriteJSON(w, statusCode, map[string]interface{}{
				"message": "Failed to update API",
				"errors":  "Internal Server Error",
			})
			return
		}

		errKey := errObj.Property
		if errKey == "" {
			errKey = "error"
		}
		utilities.WriteJSON(w, statusCode, map[string]interface{}{
			"message": "Failed to update API",
			"errors":  map[string]string{errKey: errObj.MessageToSend},
		})
		return
	}

	utilities.Log(reqID, utilities.DEBUG, path, "apiController.updateCtrl", nil, "success", nil)
	utilities.WriteJSON(w, http.StatusOK, map[string]string{
		"message": "API updated successfully",
	})
}

func (c apiController) getAllCtrl(w http.ResponseWriter, r *http.Request) {
	reqID := utilities.GetReqID(r.Context())
	path := r.URL.Path

	pageStr := r.URL.Query().Get("paginationPage")
	rowsStr := r.URL.Query().Get("paginationRows")
	sortField := r.URL.Query().Get("sortField")
	sortOrder := r.URL.Query().Get("sortOrder")
	search := r.URL.Query().Get("search")

	page := 1
	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil {
			page = p
		}
	}

	rows := 10
	if rowsStr != "" {
		if rs, err := strconv.Atoi(rowsStr); err == nil {
			rows = rs
		}
	}

	if sortField == "" {
		sortField = "created_at"
	}
	if sortOrder == "" {
		sortOrder = "DESC"
	}

	req := models.GetAllApiRequest{
		Page:      page,
		Rows:      rows,
		SortField: sortField,
		SortOrder: sortOrder,
		Search:    search,
	}

	// Logging request payload
	utilities.Log(reqID, utilities.DEBUG, path, "apiController.getAllCtrl", req, "incoming request", nil)

	if err := validate.Struct(req); err != nil {
		errMap := errUtility.ParseError(err)
		utilities.Log(reqID, utilities.WARN, path, "apiController.getAllCtrl", req, "validation failed", errMap)
		utilities.WriteJSON(w, http.StatusBadRequest, map[string]interface{}{
			"message": "Validation failed",
			"errors":  errMap,
		})
		return
	}

	list, count, errObj := c.service.GetAllSrvc(r.Context(), req)
	if errObj.Code > 0 {
		errObj.Compile()
		statusCode := int(errObj.Code)

		logLevel := utilities.ERROR
		if statusCode < 500 {
			logLevel = utilities.WARN
		}
		utilities.Log(reqID, logLevel, path, "apiController.getAllCtrl", req, "service execution failed", errObj)

		if statusCode == 500 {
			utilities.WriteJSON(w, statusCode, map[string]interface{}{
				"message": "Failed to get list api",
				"errors":  "Internal Server Error",
			})
			return
		} else {
			errKey := errObj.Property
			if errKey == "" {
				errKey = "error"
			}
			utilities.WriteJSON(w, statusCode, map[string]interface{}{
				"message": "Failed to get list api",
				"errors": map[string]string{
					errKey: errObj.MessageToSend,
				},
			})
			return
		}
	}

	utilities.Log(reqID, utilities.DEBUG, path, "apiController.getAllCtrl", nil, "success", map[string]interface{}{"count": count})
	if list == nil {
		list = []models.ApiResponse{}
	}
	utilities.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"data": map[string]interface{}{
			"listData": list,
			"count":    count,
		},
		"message": "Success get list api",
	})
}

func (c apiController) GetAllCtrl(w http.ResponseWriter, r *http.Request) {
	c.getAllCtrl(w, r)
}

var controller apiController

func factoryApiController(repo repoPostgres.ApiPostgresRepo) apiController {
	if controller.service == nil {
		controller = apiController{
			service: factoryApiService(repo),
		}
	}
	return controller
}
