package handlers_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/smakimka/balb/internal/model"
	"github.com/smakimka/balb/internal/server/handlers"
	"github.com/smakimka/balb/internal/server/storage"
	mock_storage "github.com/smakimka/balb/internal/server/storage/mock"
)

func TestSubscribe(t *testing.T) {
	type want struct {
		contentType string
		code        int
		body        model.Response
	}
	type mock struct {
		expect    bool
		returnErr error
	}
	tests := []struct {
		name        string
		method      string
		contentType string
		body        model.SubscriptionData
		mock        mock
		want        want
	}{
		{
			name:        "happy path",
			method:      http.MethodPost,
			contentType: "application/json",
			body: model.SubscriptionData{
				Front:         model.TelegramFront,
				SubscriberUID: "test_user_1",
				UserUID:       "test_user_2",
			},
			mock: mock{
				expect:    true,
				returnErr: nil,
			},
			want: want{
				contentType: "application/json",
				code:        http.StatusOK,
				body:        model.Response{},
			},
		},
		{
			name:        "subscription already exists",
			method:      http.MethodPost,
			contentType: "application/json",
			body: model.SubscriptionData{
				Front:         model.TelegramFront,
				SubscriberUID: "test_user_1",
				UserUID:       "test_user_2",
			},
			mock: mock{
				expect:    true,
				returnErr: storage.ErrSubscriptionAlreadyExists,
			},
			want: want{
				contentType: "application/json",
				code:        http.StatusBadRequest,
				body:        model.Response{Msg: "subscription already exists"},
			},
		},
		{
			name:        "wrong content type",
			method:      http.MethodPost,
			contentType: "application/xml",
			body: model.SubscriptionData{
				Front:         model.TelegramFront,
				SubscriberUID: "test_user_1",
				UserUID:       "test_user_2",
			},
			mock: mock{
				expect:    false,
				returnErr: nil,
			},
			want: want{
				contentType: "application/json",
				code:        http.StatusBadRequest,
				body:        model.Response{Msg: "wrong json"},
			},
		},
		{
			name:        "empty sub UID",
			method:      http.MethodPost,
			contentType: "application/json",
			body: model.SubscriptionData{
				Front:         model.TelegramFront,
				SubscriberUID: "",
				UserUID:       "test_user_2",
			},
			mock: mock{
				expect:    false,
				returnErr: nil,
			},
			want: want{
				contentType: "application/json",
				code:        http.StatusBadRequest,
				body:        model.Response{Msg: "wrong json"},
			},
		},
		{
			name:        "empty user UID",
			method:      http.MethodPost,
			contentType: "application/json",
			body: model.SubscriptionData{
				Front:         model.TelegramFront,
				SubscriberUID: "test_user_1",
				UserUID:       "",
			},
			mock: mock{
				expect:    false,
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
			body: model.SubscriptionData{
				Front:         model.TelegramFront,
				SubscriberUID: "test_user_1",
				UserUID:       "test_user_2",
			},
			mock: mock{
				expect:    true,
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

	ts := httptest.NewServer(getTestSubscribeRouter(m))

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.mock.expect {
				m.EXPECT().Subscribe(gomock.Any(), gomock.Eq(&test.body)).Times(1).Return(test.mock.returnErr)
			} else {
				m.EXPECT().Subscribe(gomock.Any(), gomock.Any()).Times(0)
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

func getTestSubscribeRouter(s storage.Storage) chi.Router {
	subscribeHandler := handlers.NewSubscribeHandler(s)

	r := chi.NewRouter()
	r.Post("/", subscribeHandler.ServeHTTP)

	return r
}
