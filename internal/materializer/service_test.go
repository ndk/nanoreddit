package materializer

import (
	"context"
	"errors"
	"testing"

	"github.com/go-redis/redis/v8"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/mock"

	"nanoreddit/internal/storage"
)

type mockRedis struct {
	redis.Cmdable

	m *mock.Mock
}

func (m *mockRedis) XReadGroup(ctx context.Context, a *redis.XReadGroupArgs) *redis.XStreamSliceCmd {
	args := m.m.Called(ctx, a)
	return args.Get(0).(*redis.XStreamSliceCmd)
}

func (m *mockRedis) LPush(ctx context.Context, key string, values ...interface{}) *redis.IntCmd {
	args := m.m.Called(ctx, key, values)
	return args.Get(0).(*redis.IntCmd)
}

func (m *mockRedis) ZAdd(ctx context.Context, key string, members ...*redis.Z) *redis.IntCmd {
	args := m.m.Called(ctx, key, members)
	return args.Get(0).(*redis.IntCmd)
}

func TestService(t *testing.T) {
	Convey("Test materializer", t, func() {
		m := &mock.Mock{}
		srv := service{
			cfg:    &Config{},
			client: &mockRedis{m: m},
		}

		Convey("Execute", func() {
			Convey("Suppress safe errors", func() {
				m.
					On("XReadGroup", mock.Anything, mock.Anything).
					Return(redis.NewXStreamSliceCmdResult(nil, errors.New("error")))

				err := srv.Execute()

				So(err, ShouldBeError)
				So(err, ShouldNotEqual, `Couldn't create a stream: error`)
			})

			Convey("It fails if XReadGroup has been failed", func() {
				m.
					On("XGroupCreateMkStream", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(redis.NewStatusResult("", nil)).
					On("XReadGroup", mock.Anything, mock.Anything).
					Return(redis.NewXStreamSliceCmdResult(nil, errors.New("error")))

				err := srv.Execute()

				So(err.Error(), ShouldEqual, `couldn't read a group: error`)
			})

			Convey("It fails if XReadGroup has returned an inproper number of streams", func() {
				m.
					On("XReadGroup", mock.Anything, mock.Anything).
					Return(redis.NewXStreamSliceCmdResult([]redis.XStream{{}, {}}, nil))

				err := srv.Execute()

				So(err.Error(), ShouldEqual, `unexpected number of streams: 2`)
			})

			Convey("It keeps fetching", func() {
				m.
					On("XReadGroup", mock.Anything, mock.Anything).
					Return(redis.NewXStreamSliceCmdResult([]redis.XStream{{}}, nil)).Once().
					On("XReadGroup", mock.Anything, mock.Anything).
					Return(redis.NewXStreamSliceCmdResult([]redis.XStream{{}}, errors.New("stop")))

				err := srv.Execute()

				So(err.Error(), ShouldEqual, `couldn't read a group: stop`)
			})

			Convey("It fails if a message doesn't have an event payload", func() {
				m.
					On("XReadGroup", mock.Anything, mock.Anything).
					Return(redis.NewXStreamSliceCmdResult(
						[]redis.XStream{
							{Messages: []redis.XMessage{
								{Values: map[string]interface{}{}},
							},
							},
						}, nil)).Once()

				err := srv.Execute()

				So(err.Error(), ShouldEqual, `couldn't find an event in a message: { map[]}`)
			})

			Convey("It fails if an event payload is undecryptable", func() {
				m.
					On("XReadGroup", mock.Anything, mock.Anything).
					Return(redis.NewXStreamSliceCmdResult(
						[]redis.XStream{
							{Messages: []redis.XMessage{
								{Values: map[string]interface{}{storage.StreamValueField: ""}},
							},
							},
						}, nil)).Once()

				err := srv.Execute()

				So(err.Error(), ShouldEqual, `couldn't unmarshal a saved post: unexpected end of JSON input`)
			})

			Convey("A promoted post", func() {
				Convey("It fails if a promotion event cannot be save on the storage", func() {
					m.
						On("XReadGroup", mock.Anything, mock.Anything).
						Return(redis.NewXStreamSliceCmdResult(
							[]redis.XStream{
								{Messages: []redis.XMessage{
									{Values: map[string]interface{}{storage.StreamValueField: `{"promoted": true}`}},
								},
								},
							}, nil)).Once().
						On("LPush", mock.Anything, mock.Anything, mock.Anything).
						Return(redis.NewIntResult(123, errors.New("error")))

					err := srv.Execute()

					So(err.Error(), ShouldEqual, `couldn't put a promoted post into a ring: error`)
				})

				Convey("Successful story", func() {
					m.
						On("XReadGroup", mock.Anything, mock.Anything).
						Return(redis.NewXStreamSliceCmdResult(
							[]redis.XStream{
								{Messages: []redis.XMessage{
									{Values: map[string]interface{}{storage.StreamValueField: `{"promoted": true}`}},
								},
								},
							}, nil)).Once().
						On("LPush", mock.Anything, mock.Anything, mock.Anything).
						Return(redis.NewIntResult(123, nil)).Once().
						On("XReadGroup", mock.Anything, mock.Anything).
						Return(redis.NewXStreamSliceCmdResult([]redis.XStream{{}}, errors.New("stop")))

					err := srv.Execute()

					So(err.Error(), ShouldEqual, `couldn't read a group: stop`)
				})
			})

			Convey("An ordinary post", func() {
				Convey("It fails if an event cannot be save on the storage", func() {
					m.
						On("XReadGroup", mock.Anything, mock.Anything).
						Return(redis.NewXStreamSliceCmdResult(
							[]redis.XStream{
								{Messages: []redis.XMessage{
									{Values: map[string]interface{}{storage.StreamValueField: `{"promoted": false}`}},
								},
								},
							}, nil)).Once().
						On("ZAdd", mock.Anything, mock.Anything, mock.Anything).
						Return(redis.NewIntResult(123, errors.New("error")))

					err := srv.Execute()

					So(err.Error(), ShouldEqual, `couldn't put a post into the feed: error`)
				})

				Convey("Successful story", func() {
					m.
						On("XReadGroup", mock.Anything, mock.Anything).
						Return(redis.NewXStreamSliceCmdResult(
							[]redis.XStream{
								{Messages: []redis.XMessage{
									{Values: map[string]interface{}{storage.StreamValueField: `{"promoted": false}`}},
								},
								},
							}, nil)).Once().
						On("ZAdd", mock.Anything, mock.Anything, mock.Anything).
						Return(redis.NewIntResult(123, nil)).Once().
						On("XReadGroup", mock.Anything, mock.Anything).
						Return(redis.NewXStreamSliceCmdResult([]redis.XStream{{}}, errors.New("stop")))

					err := srv.Execute()

					So(err.Error(), ShouldEqual, `couldn't read a group: stop`)
				})
			})

			Convey("Successful story (mixed messages)", func() {
				m.
					On("XReadGroup", mock.Anything, mock.Anything).
					Return(redis.NewXStreamSliceCmdResult(
						[]redis.XStream{
							{Messages: []redis.XMessage{
								{Values: map[string]interface{}{storage.StreamValueField: `{"promoted": false}`}},
								{Values: map[string]interface{}{storage.StreamValueField: `{"promoted": true}`}},
								{Values: map[string]interface{}{storage.StreamValueField: `{"promoted": false}`}},
								{Values: map[string]interface{}{storage.StreamValueField: `{"promoted": true}`}},
							},
							},
						}, nil)).Once().
					On("ZAdd", mock.Anything, mock.Anything, mock.Anything).
					Return(redis.NewIntResult(123, nil)).Once().
					On("LPush", mock.Anything, mock.Anything, mock.Anything).
					Return(redis.NewIntResult(123, nil)).Once().
					On("ZAdd", mock.Anything, mock.Anything, mock.Anything).
					Return(redis.NewIntResult(123, nil)).Once().
					On("LPush", mock.Anything, mock.Anything, mock.Anything).
					Return(redis.NewIntResult(123, nil)).Once().
					On("XReadGroup", mock.Anything, mock.Anything).
					Return(redis.NewXStreamSliceCmdResult([]redis.XStream{{}}, errors.New("stop")))

				err := srv.Execute()

				So(err.Error(), ShouldEqual, `couldn't read a group: stop`)
			})
		})
	})
}
