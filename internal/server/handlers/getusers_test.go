package handlers_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/smakimka/balb/internal/model"
	"github.com/smakimka/balb/internal/server/handlers"
	"github.com/smakimka/balb/internal/server/storage"
	mock_storage "github.com/smakimka/balb/internal/server/storage/mock"
)

func TestGetUsers(t *testing.T) {
	birthday := time.Date(2001, 02, 24, 0, 0, 0, 0, time.UTC)

	type want struct {
		contentType string
		code        int
		response    model.Response
		users       []model.User
	}
	type mock struct {
		expect      bool
		returnUsers []model.User
		returnErr   error
	}
	tests := []struct {
		name   string
		method string
		front  int
		mock   mock
		want   want
	}{
		{
			name:   "happy path",
			method: http.MethodGet,
			front:  model.TelegramFront,
			mock: mock{
				expect: true,
				returnUsers: []model.User{model.User{
					ID:       1,
					Front:    model.TelegramFront,
					UID:      "test_user",
					FIO:      "t.t.",
					Birthday: birthday,
				}},
				returnErr: nil,
			},
			want: want{
				contentType: "application/json",
				code:        http.StatusOK,
				response:    model.Response{},
				users: []model.User{model.User{
					ID:       1,
					Front:    model.TelegramFront,
					UID:      "test_user",
					FIO:      "t.t.",
					Birthday: birthday,
				}},
			},
		},
		{
			name:   "storage error",
			method: http.MethodGet,
			front:  model.TelegramFront,
			mock: mock{
				expect:      true,
				returnUsers: []model.User{},
				returnErr:   errors.New("some storage error"),
			},
			want: want{
				contentType: "application/json",
				code:        http.StatusInternalServerError,
				response:    model.Response{Msg: "internal server error"},
				users:       []model.User{},
			},
		},
	}

	ctrl := gomock.NewController(t)
	m := mock_storage.NewMockStorage(ctrl)

	ts := httptest.NewServer(getTestGetUsersRouter(m))

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.mock.expect {
				m.EXPECT().GetUsers(gomock.Any(), gomock.Eq(test.front)).Times(1).Return(test.mock.returnUsers, test.mock.returnErr)
			} else {
				m.EXPECT().GetUsers(gomock.Any(), gomock.Any()).Times(0)
			}

			req, err := http.NewRequest(test.method, fmt.Sprintf("%s/%d", ts.URL, test.front), nil)
			require.NoError(t, err)

			resp, err := ts.Client().Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			respBody, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			var respResponse model.Response
			var respUser []model.User
			if test.want.response.Msg != "" {
				err = json.Unmarshal(respBody, &respResponse)
				require.NoError(t, err)
			} else {
				err = json.Unmarshal(respBody, &respUser)
				require.NoError(t, err)
			}

			assert.Equal(t, test.want.code, resp.StatusCode)
			assert.Equal(t, test.want.contentType, resp.Header.Get("Content-Type"))
			if test.want.response.Msg != "" {
				assert.Equal(t, test.want.response, respResponse)
			} else {
				assert.Equal(t, test.want.users, respUser)
			}
		})
	}
}

func getTestGetUsersRouter(s storage.Storage) chi.Router {
	addUsersHandler := handlers.NewGetUsersHandler(s)

	r := chi.NewRouter()
	r.Get("/{front}", addUsersHandler.ServeHTTP)

	return r
}
