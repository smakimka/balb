package handlers_test

import (
	"bytes"
	"encoding/json"
	"errors"
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

func TestAddUser(t *testing.T) {
	birtday := time.Date(2001, 2, 24, 0, 0, 0, 0, time.UTC)

	type want struct {
		contentType string
		code        int
		body        model.Response
	}
	type mock struct {
		expect    bool
		returnID  int
		returnErr error
	}
	tests := []struct {
		name        string
		method      string
		contentType string
		body        model.User
		mock        mock
		want        want
	}{
		{
			name:        "happy path",
			method:      http.MethodPost,
			contentType: "application/json",
			body: model.User{
				Birthday: birtday,
				Front:    model.TelegramFront,
				UID:      "test_user",
				FIO:      "test_fio",
				Wishlist: "test_wishlist",
			},
			mock: mock{
				expect:    true,
				returnID:  1,
				returnErr: nil,
			},
			want: want{
				contentType: "application/json",
				code:        http.StatusOK,
				body:        model.Response{},
			},
		},
		{
			name:        "user already exists",
			method:      http.MethodPost,
			contentType: "application/json",
			body: model.User{
				Birthday: birtday,
				Front:    model.TelegramFront,
				UID:      "test_user",
				FIO:      "test_fio",
				Wishlist: "test_wishlist",
			},
			mock: mock{
				expect:    true,
				returnID:  0,
				returnErr: storage.ErrUserAlreadyExists,
			},
			want: want{
				contentType: "application/json",
				code:        http.StatusBadRequest,
				body:        model.Response{Msg: "user already exists"},
			},
		},
		{
			name:        "wrong content type",
			method:      http.MethodPost,
			contentType: "application/xml",
			body: model.User{
				Birthday: birtday,
				Front:    model.TelegramFront,
				UID:      "test_user",
				FIO:      "test_fio",
				Wishlist: "test_wishlist",
			},
			mock: mock{
				expect:    false,
				returnID:  1,
				returnErr: nil,
			},
			want: want{
				contentType: "application/json",
				code:        http.StatusBadRequest,
				body:        model.Response{Msg: "wrong json"},
			},
		},
		{
			name:        "empty UID",
			method:      http.MethodPost,
			contentType: "application/json",
			body: model.User{
				Birthday: birtday,
				Front:    model.TelegramFront,
				UID:      "",
				FIO:      "test_fio",
				Wishlist: "test_wishlist",
			},
			mock: mock{
				expect:    false,
				returnID:  1,
				returnErr: nil,
			},
			want: want{
				contentType: "application/json",
				code:        http.StatusBadRequest,
				body:        model.Response{Msg: "wrong json"},
			},
		},
		{
			name:        "sql error",
			method:      http.MethodPost,
			contentType: "application/json",
			body: model.User{
				Birthday: birtday,
				Front:    model.TelegramFront,
				UID:      "test_user",
				FIO:      "test_fio",
				Wishlist: "test_wishlist",
			},
			mock: mock{
				expect:    true,
				returnID:  0,
				returnErr: errors.New("postgres err"),
			},
			want: want{
				contentType: "application/json",
				code:        http.StatusInternalServerError,
				body:        model.Response{Msg: "internal server error"},
			},
		},
	}

	ctrl := gomock.NewController(t)
	m := mock_storage.NewMockStorage(ctrl)

	ts := httptest.NewServer(getTestAddUserRouter(m))

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.mock.expect {
				m.EXPECT().CreateUser(gomock.Any(), gomock.Eq(&test.body)).Times(1).Return(test.mock.returnID, test.mock.returnErr)
			} else {
				m.EXPECT().CreateUser(gomock.Any(), gomock.Any()).Times(0)
			}

			reqBody, err := json.Marshal(test.body)
			require.NoError(t, err)

			req, err := http.NewRequest(test.method, ts.URL, bytes.NewReader(reqBody))
			require.NoError(t, err)

			req.Header.Add("Content-type", test.contentType)

			resp, err := ts.Client().Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			respBody, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			var respData model.Response
			err = json.Unmarshal(respBody, &respData)
			require.NoError(t, err)

			assert.Equal(t, test.want.code, resp.StatusCode)
			assert.Equal(t, test.want.contentType, resp.Header.Get("Content-Type"))
			assert.Equal(t, test.want.body, respData)
		})
	}
}

func getTestAddUserRouter(s storage.Storage) chi.Router {
	addUserHandler := handlers.NewAdduserHandler(s)

	r := chi.NewRouter()
	r.Post("/", addUserHandler.ServeHTTP)

	return r
}
