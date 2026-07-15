package tests

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	_ "github.com/lib/pq"

	"github.com/IDTS-LAB/go-codebase/internal/shared/cqrs"
	"github.com/IDTS-LAB/go-codebase/internal/shared/events"
	"github.com/IDTS-LAB/go-codebase/internal/shared/tenantfilter"
	"github.com/IDTS-LAB/go-codebase/internal/shared/validator"
	"github.com/IDTS-LAB/go-codebase/internal/testhelper"
	"github.com/IDTS-LAB/go-codebase/internal/todo/application/command"
	"github.com/IDTS-LAB/go-codebase/internal/todo/application/dto"
	"github.com/IDTS-LAB/go-codebase/internal/todo/application/query"
	"github.com/IDTS-LAB/go-codebase/internal/todo/domain/service"
	todoPersistence "github.com/IDTS-LAB/go-codebase/internal/todo/infrastructure/persistence"
	httpHandler "github.com/IDTS-LAB/go-codebase/internal/todo/interfaces/http"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func integrationHandler(t *testing.T) *httpHandler.Handler {
	t.Helper()
	repo := todoPersistence.NewTodoRepository(db, &tenantfilter.Config{})
	domainSvc := service.NewTodoDomainService(repo)
	eventBus := events.NewInMemoryEventBus()

	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()

	cmdBus.Register(command.CreateTodoCommand{}, command.NewCreateTodoHandler(domainSvc, eventBus))
	cmdBus.Register(command.UpdateTodoCommand{}, command.NewUpdateTodoHandler(domainSvc, eventBus))
	cmdBus.Register(command.DeleteTodoCommand{}, command.NewDeleteTodoHandler(domainSvc, eventBus))
	cmdBus.Register(command.CompleteTodoCommand{}, command.NewCompleteTodoHandler(domainSvc, eventBus))

	qBus.Register(query.GetTodoQuery{}, query.NewGetTodoHandler(domainSvc))
	qBus.Register(query.ListTodosQuery{}, query.NewListTodosHandler(domainSvc))
	qBus.Register(query.SearchTodosQuery{}, query.NewSearchTodosHandler(domainSvc))

	v := validator.New()
	return httpHandler.NewHandler(cmdBus, qBus, v)
}

func TestIntegration_CreateTodoAndGetTodo(t *testing.T) {
	testhelper.WithTx(t, db, func(tx *sql.Tx) {
		h := integrationHandler(t)

		uniqueTitle := "BuyMilk_" + uuid.New().String()
		body, _ := json.Marshal(dto.CreateTodoRequest{Title: uniqueTitle, Description: "2% milk"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/todos", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		h.CreateTodo(rr, req)

		assert.Equal(t, http.StatusCreated, rr.Code)
		var createResp map[string]interface{}
		json.NewDecoder(rr.Body).Decode(&createResp)
		assert.True(t, createResp["success"].(bool))
		createData := createResp["data"].(map[string]interface{})
		todoID := createData["id"].(string)
		assert.Equal(t, uniqueTitle, createData["title"])
		assert.Equal(t, "2% milk", createData["description"])
		assert.Equal(t, false, createData["completed"])
		assert.NotEmpty(t, createData["created_at"])
		assert.NotEmpty(t, createData["updated_at"])

		getReq := httptest.NewRequest(http.MethodGet, "/api/v1/todos/"+todoID, nil)
		getReq = withChiContext(getReq, map[string]string{"id": todoID})
		getRR := httptest.NewRecorder()
		h.GetTodo(getRR, getReq)

		assert.Equal(t, http.StatusOK, getRR.Code)
		var getResp map[string]interface{}
		json.NewDecoder(getRR.Body).Decode(&getResp)
		assert.True(t, getResp["success"].(bool))
		assert.Nil(t, getResp["meta"])
		getData := getResp["data"].(map[string]interface{})
		assert.Equal(t, todoID, getData["id"])
		assert.Equal(t, uniqueTitle, getData["title"])
		assert.Equal(t, "2% milk", getData["description"])
		assert.Equal(t, false, getData["completed"])
	})
}

func TestIntegration_CreateTodoAndListTodos(t *testing.T) {
	testhelper.WithTx(t, db, func(tx *sql.Tx) {
		h := integrationHandler(t)

		uniqueTitle1 := "ListFirst_" + uuid.New().String()
		body1, _ := json.Marshal(dto.CreateTodoRequest{Title: uniqueTitle1, Description: "First"})
		req1 := httptest.NewRequest(http.MethodPost, "/api/v1/todos", bytes.NewReader(body1))
		req1.Header.Set("Content-Type", "application/json")
		rr1 := httptest.NewRecorder()
		h.CreateTodo(rr1, req1)
		assert.Equal(t, http.StatusCreated, rr1.Code)
		var create1Resp map[string]interface{}
		json.NewDecoder(rr1.Body).Decode(&create1Resp)
		id1 := create1Resp["data"].(map[string]interface{})["id"].(string)

		uniqueTitle2 := "ListSecond_" + uuid.New().String()
		body2, _ := json.Marshal(dto.CreateTodoRequest{Title: uniqueTitle2, Description: "Second"})
		req2 := httptest.NewRequest(http.MethodPost, "/api/v1/todos", bytes.NewReader(body2))
		req2.Header.Set("Content-Type", "application/json")
		rr2 := httptest.NewRecorder()
		h.CreateTodo(rr2, req2)
		assert.Equal(t, http.StatusCreated, rr2.Code)
		var create2Resp map[string]interface{}
		json.NewDecoder(rr2.Body).Decode(&create2Resp)
		id2 := create2Resp["data"].(map[string]interface{})["id"].(string)

		listReq := httptest.NewRequest(http.MethodGet, "/api/v1/todos", nil)
		listRR := httptest.NewRecorder()
		h.ListTodos(listRR, listReq)

		assert.Equal(t, http.StatusOK, listRR.Code)
		var listResp map[string]interface{}
		json.NewDecoder(listRR.Body).Decode(&listResp)
		assert.True(t, listResp["success"].(bool))
		assert.NotNil(t, listResp["meta"])

		listData := listResp["data"].([]interface{})
		foundIDs := make([]string, len(listData))
		for i, item := range listData {
			foundIDs[i] = item.(map[string]interface{})["id"].(string)
		}
		assert.Contains(t, foundIDs, id1)
		assert.Contains(t, foundIDs, id2)
	})
}

func TestIntegration_CreateTodoAndUpdateTodo(t *testing.T) {
	testhelper.WithTx(t, db, func(tx *sql.Tx) {
		h := integrationHandler(t)

		uniqueTitle := "OldTitle_" + uuid.New().String()
		body, _ := json.Marshal(dto.CreateTodoRequest{Title: uniqueTitle, Description: "Old Desc"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/todos", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		h.CreateTodo(rr, req)
		assert.Equal(t, http.StatusCreated, rr.Code)
		var createResp map[string]interface{}
		json.NewDecoder(rr.Body).Decode(&createResp)
		todoID := createResp["data"].(map[string]interface{})["id"].(string)

		updatedTitle := "UpdatedTitle_" + uuid.New().String()
		updateBody, _ := json.Marshal(dto.UpdateTodoRequest{Title: updatedTitle, Description: "Updated Desc"})
		updateReq := httptest.NewRequest(http.MethodPut, "/api/v1/todos/"+todoID, bytes.NewReader(updateBody))
		updateReq.Header.Set("Content-Type", "application/json")
		updateReq = withChiContext(updateReq, map[string]string{"id": todoID})
		updateRR := httptest.NewRecorder()
		h.UpdateTodo(updateRR, updateReq)
		assert.Equal(t, http.StatusOK, updateRR.Code)

		getReq := httptest.NewRequest(http.MethodGet, "/api/v1/todos/"+todoID, nil)
		getReq = withChiContext(getReq, map[string]string{"id": todoID})
		getRR := httptest.NewRecorder()
		h.GetTodo(getRR, getReq)
		assert.Equal(t, http.StatusOK, getRR.Code)

		var getResp map[string]interface{}
		json.NewDecoder(getRR.Body).Decode(&getResp)
		getData := getResp["data"].(map[string]interface{})
		assert.Equal(t, updatedTitle, getData["title"])
		assert.Equal(t, "Updated Desc", getData["description"])
	})
}

func TestIntegration_CreateTodoAndDeleteTodo(t *testing.T) {
	testhelper.WithTx(t, db, func(tx *sql.Tx) {
		h := integrationHandler(t)

		uniqueTitle := "DeleteMe_" + uuid.New().String()
		body, _ := json.Marshal(dto.CreateTodoRequest{Title: uniqueTitle, Description: "To be deleted"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/todos", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		h.CreateTodo(rr, req)
		assert.Equal(t, http.StatusCreated, rr.Code)
		var createResp map[string]interface{}
		json.NewDecoder(rr.Body).Decode(&createResp)
		todoID := createResp["data"].(map[string]interface{})["id"].(string)

		delReq := httptest.NewRequest(http.MethodDelete, "/api/v1/todos/"+todoID, nil)
		delReq = withChiContext(delReq, map[string]string{"id": todoID})
		delRR := httptest.NewRecorder()
		h.DeleteTodo(delRR, delReq)
		assert.Equal(t, http.StatusOK, delRR.Code)

		getReq := httptest.NewRequest(http.MethodGet, "/api/v1/todos/"+todoID, nil)
		getReq = withChiContext(getReq, map[string]string{"id": todoID})
		getRR := httptest.NewRecorder()
		h.GetTodo(getRR, getReq)
		assert.Equal(t, http.StatusNotFound, getRR.Code)

		var getResp map[string]interface{}
		json.NewDecoder(getRR.Body).Decode(&getResp)
		assert.False(t, getResp["success"].(bool))
		assert.Nil(t, getResp["data"])
		assert.Equal(t, "NOT_FOUND", getResp["error"].(map[string]interface{})["code"])
	})
}

func TestIntegration_CreateTodoAndCompleteTodo(t *testing.T) {
	testhelper.WithTx(t, db, func(tx *sql.Tx) {
		h := integrationHandler(t)

		uniqueTitle := "CompleteMe_" + uuid.New().String()
		body, _ := json.Marshal(dto.CreateTodoRequest{Title: uniqueTitle, Description: ""})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/todos", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		h.CreateTodo(rr, req)
		assert.Equal(t, http.StatusCreated, rr.Code)
		var createResp map[string]interface{}
		json.NewDecoder(rr.Body).Decode(&createResp)
		todoID := createResp["data"].(map[string]interface{})["id"].(string)
		assert.Equal(t, uniqueTitle, createResp["data"].(map[string]interface{})["title"])
		assert.Equal(t, false, createResp["data"].(map[string]interface{})["completed"])

		completeReq := httptest.NewRequest(http.MethodPatch, "/api/v1/todos/"+todoID+"/complete", nil)
		completeReq = withChiContext(completeReq, map[string]string{"id": todoID})
		completeRR := httptest.NewRecorder()
		h.CompleteTodo(completeRR, completeReq)
		assert.Equal(t, http.StatusOK, completeRR.Code)

		var completeResp map[string]interface{}
		json.NewDecoder(completeRR.Body).Decode(&completeResp)
		assert.True(t, completeResp["success"].(bool))
		assert.Equal(t, true, completeResp["data"].(map[string]interface{})["completed"])

		getReq := httptest.NewRequest(http.MethodGet, "/api/v1/todos/"+todoID, nil)
		getReq = withChiContext(getReq, map[string]string{"id": todoID})
		getRR := httptest.NewRecorder()
		h.GetTodo(getRR, getReq)
		assert.Equal(t, http.StatusOK, getRR.Code)

		var getResp map[string]interface{}
		json.NewDecoder(getRR.Body).Decode(&getResp)
		assert.Equal(t, true, getResp["data"].(map[string]interface{})["completed"])
	})
}

func TestIntegration_CreateTodoAndSearchTodos(t *testing.T) {
	testhelper.WithTx(t, db, func(tx *sql.Tx) {
		h := integrationHandler(t)

		uniqueTitle := "SearchableTodo_" + uuid.New().String()
		body, _ := json.Marshal(dto.CreateTodoRequest{Title: uniqueTitle, Description: "Searchable desc"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/todos", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		h.CreateTodo(rr, req)
		assert.Equal(t, http.StatusCreated, rr.Code)
		var createResp map[string]interface{}
		json.NewDecoder(rr.Body).Decode(&createResp)
		todoID := createResp["data"].(map[string]interface{})["id"].(string)

		searchReq := httptest.NewRequest(http.MethodGet, "/api/v1/todos/search?q="+uniqueTitle, nil)
		searchRR := httptest.NewRecorder()
		h.SearchTodos(searchRR, searchReq)

		assert.Equal(t, http.StatusOK, searchRR.Code)
		var searchResp map[string]interface{}
		json.NewDecoder(searchRR.Body).Decode(&searchResp)
		assert.True(t, searchResp["success"].(bool))
		assert.NotNil(t, searchResp["meta"])

		searchData := searchResp["data"].([]interface{})
		if assert.Equal(t, 1, len(searchData)) {
			assert.Equal(t, todoID, searchData[0].(map[string]interface{})["id"].(string))
		}
	})
}

func TestIntegration_CreateTodo_InvalidBody(t *testing.T) {
	testhelper.WithTx(t, db, func(tx *sql.Tx) {
		h := integrationHandler(t)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/todos", bytes.NewReader([]byte("not valid json")))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		h.CreateTodo(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		var resp map[string]interface{}
		json.NewDecoder(rr.Body).Decode(&resp)
		assert.False(t, resp["success"].(bool))
		assert.Nil(t, resp["data"])
		assert.Equal(t, "VALIDATION_ERROR", resp["error"].(map[string]interface{})["code"])
	})
}

func TestIntegration_GetTodo_NotFound(t *testing.T) {
	testhelper.WithTx(t, db, func(tx *sql.Tx) {
		h := integrationHandler(t)
		nonExistentID := uuid.New().String()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/todos/"+nonExistentID, nil)
		req = withChiContext(req, map[string]string{"id": nonExistentID})
		rr := httptest.NewRecorder()
		h.GetTodo(rr, req)

		assert.Equal(t, http.StatusNotFound, rr.Code)
		var resp map[string]interface{}
		json.NewDecoder(rr.Body).Decode(&resp)
		assert.False(t, resp["success"].(bool))
		assert.Nil(t, resp["data"])
		assert.Equal(t, "NOT_FOUND", resp["error"].(map[string]interface{})["code"])
	})
}

func TestIntegration_CompleteTodo_AlreadyDone(t *testing.T) {
	testhelper.WithTx(t, db, func(tx *sql.Tx) {
		h := integrationHandler(t)

		uniqueTitle := "AlreadyDone_" + uuid.New().String()
		body, _ := json.Marshal(dto.CreateTodoRequest{Title: uniqueTitle, Description: ""})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/todos", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		h.CreateTodo(rr, req)
		assert.Equal(t, http.StatusCreated, rr.Code)
		var createResp map[string]interface{}
		json.NewDecoder(rr.Body).Decode(&createResp)
		todoID := createResp["data"].(map[string]interface{})["id"].(string)

		completeReq1 := httptest.NewRequest(http.MethodPatch, "/api/v1/todos/"+todoID+"/complete", nil)
		completeReq1 = withChiContext(completeReq1, map[string]string{"id": todoID})
		completeRR1 := httptest.NewRecorder()
		h.CompleteTodo(completeRR1, completeReq1)
		assert.Equal(t, http.StatusOK, completeRR1.Code)

		completeReq2 := httptest.NewRequest(http.MethodPatch, "/api/v1/todos/"+todoID+"/complete", nil)
		completeReq2 = withChiContext(completeReq2, map[string]string{"id": todoID})
		completeRR2 := httptest.NewRecorder()
		h.CompleteTodo(completeRR2, completeReq2)

		assert.Equal(t, http.StatusConflict, completeRR2.Code)
		var resp map[string]interface{}
		json.NewDecoder(completeRR2.Body).Decode(&resp)
		assert.False(t, resp["success"].(bool))
		assert.Nil(t, resp["data"])
		assert.Equal(t, "CONFLICT", resp["error"].(map[string]interface{})["code"])
	})
}
