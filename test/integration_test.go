//  +build integration

package integration

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"testing"

	"github.com/go-redis/redis/v8"
	"github.com/go-resty/resty/v2"
	"github.com/smartystreets/assertions"
	. "github.com/smartystreets/goconvey/convey"

	"nanoreddit/pkg/protocol"
)

// func TestMain(m *testing.M) {
// 	convey.SuppressConsoleStatistics()

// 	pool, err := dockertest.NewPool("")
// 	if err != nil {
// 		log.Fatalf("Could not connect to docker: %s", err)
// 	}
// 	pool.MaxWait = 10 * time.Second

// 	resource, err := pool.RunWithOptions(
// 		&dockertest.RunOptions{
// 			Repository: "redis",
// 			Tag:        "latest",
// 			// PortBindings: map[docker.Port][]docker.PortBinding{
// 			// 	"6379": {{HostIP: "localhost", HostPort: "6379"}},
// 			// },
// 			// ExposedPorts: []string{"6379"},
// 		}, func(config *docker.HostConfig) {
// 			// set AutoRemove to true so that stopped container goes away by itself
// 			config.AutoRemove = true
// 			config.RestartPolicy = docker.RestartPolicy{
// 				Name: "no",
// 			}
// 		})
// 	if err != nil {
// 		log.Fatalf("Could not start resource: %s", err)
// 	}
// 	resource.Expire(20)

// 	defer func() {
// 		if err = pool.Purge(resource); err != nil {
// 			log.Fatalf("Could not purge resource: %s", err)
// 		}
// 	}()

// 	if err := pool.Retry(func() error {
// 		db := redis.NewClient(&redis.Options{
// 			// Addr: "localhost:6379",
// 			Addr: resource.GetHostPort("6379/tcp"),
// 		})
// 		return db.Ping(context.Background()).Err()
// 	}); err != nil {
// 		log.Fatalf("Could not connect to docker: %s", err)
// 	}

// 	result := m.Run()

// 	convey.PrintConsoleStatistics()
// 	os.Exit(result)
// }

func TestNanoreddit(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
		return
	}

	Convey("Integration test", t, func() {
		const pageSize = 25

		redisOpt, err := redis.ParseURL("redis://localhost:6379/0")
		So(err, ShouldBeNil)
		redisClient := redis.NewClient(redisOpt)
		defer func() {
			err := redisClient.Close()
			So(err, ShouldBeNil)
		}()

		ctx := context.Background()

		{
			err := redisClient.Del(ctx, "feed").Err()
			So(err, ShouldBeNil)
		}
		{
			err := redisClient.Del(ctx, "promotion").Err()
			So(err, ShouldBeNil)
		}
		{
			err := redisClient.XTrim(ctx, "posts", 0).Err()
			So(err, ShouldBeNil)
		}

		c := resty.New()
		r := c.R()

		Convey("Initially, we've got an empty feed", func() {
			var feed []protocol.Post
			resp, err := r.SetResult(&feed).Get("http://localhost:8080/feed")
			So(err, ShouldBeNil)
			So(feed, ShouldBeEmpty)
			So(resp.StatusCode(), ShouldEqual, http.StatusOK)
		})

		Convey("Posts are stored in the order sorted by their score", func() {
			var posts []protocol.Post
			direction := 1
			for pages := 1; pages <= 10; pages++ {
				for i := 0; i < 5; i++ {
					for score := 6 * direction; score != direction; score -= direction {
						post := protocol.Post{
							Author:    fmt.Sprintf("t2_%08x", len(posts)),
							Subreddit: fmt.Sprintf("subreddit %d", len(posts)),
							Title:     fmt.Sprintf("title %d", len(posts)),
							Score:     len(posts)*5 + score + 10,
						}
						{
							resp, err := r.SetBody(&post).Post("http://localhost:8080/submit")
							So(err, ShouldBeNil)
							So(string(resp.Body()), ShouldBeEmpty)
							So(resp.StatusCode(), ShouldEqual, http.StatusOK)
						}
						{
							posts = append(posts, post)
							sort.Slice(posts, func(i, j int) bool {
								return posts[i].Score > posts[j].Score
							})
						}
						for p := 0; p < pages; p++ {
							var feed []protocol.Post
							resp, err := r.SetResult(&feed).SetQueryParam("page", strconv.Itoa(p)).Get("http://localhost:8080/feed")
							So(err, ShouldBeNil)
							{
								begin, end := p*pageSize, (p+1)*pageSize
								if end > len(posts) {
									end = len(posts)
								}
								expectation := posts[begin:end]
								So(feed, assertions.ShouldResemble, expectation)
							}
							So(resp.StatusCode(), ShouldEqual, http.StatusOK)
						}
						// The next page should be empty
						{
							var feed []protocol.Post
							resp, err := r.SetResult(&feed).SetQueryParam("page", strconv.Itoa(pages)).Get("http://localhost:8080/feed")
							So(err, ShouldBeNil)
							So(feed, ShouldBeEmpty)
							So(resp.StatusCode(), ShouldEqual, http.StatusOK)
						}
					}
				}
			}
		})

		Convey("Promoted posts should not appear if not-promoted ones are too few", func() {
			var promoted []protocol.Post
			for i := 0; i < 10; i++ {
				post := protocol.Post{
					Author:    fmt.Sprintf("t2_%08x", i),
					Subreddit: fmt.Sprintf("subreddit %d", i),
					Title:     fmt.Sprintf("XXX title %d", i),
					Score:     123,
					Promoted:  true,
				}
				{
					resp, err := r.SetBody(&post).Post("http://localhost:8080/submit")
					So(err, ShouldBeNil)
					So(string(resp.Body()), ShouldBeEmpty)
					So(resp.StatusCode(), ShouldEqual, http.StatusOK)
				}
				promoted = append(promoted, post)
				{
					var feed []protocol.Post
					resp, err := r.SetResult(&feed).Get("http://localhost:8080/feed")
					So(err, ShouldBeNil)
					So(feed, ShouldBeEmpty)
					So(resp.StatusCode(), ShouldEqual, http.StatusOK)
				}
			}

			score := 999
			var posts []protocol.Post
			for pages := 1; pages <= 10; pages++ {
				for i := 0; i < pageSize; i++ {
					post := protocol.Post{
						Author:    fmt.Sprintf("t2_%08x", score),
						Subreddit: fmt.Sprintf("subreddit %d", score),
						Title:     fmt.Sprintf("title %d", score),
						Score:     score,
					}
					{
						resp, err := r.SetBody(&post).Post("http://localhost:8080/submit")
						So(err, ShouldBeNil)
						So(string(resp.Body()), ShouldBeEmpty)
						So(resp.StatusCode(), ShouldEqual, http.StatusOK)
					}
					posts = append(posts, post)
					for p := 0; p < pages; p++ {
						var feed []protocol.Post
						resp, err := r.SetResult(&feed).SetQueryParam("page", strconv.Itoa(p)).Get("http://localhost:8080/feed")
						So(err, ShouldBeNil)
						{
							begin, end := p*pageSize, (p+1)*pageSize
							if end > len(posts) {
								end = len(posts)
							}
							expectation := posts[begin:end]
							switch {
							case len(expectation) < 3:
								So(feed, assertions.ShouldResemble, expectation)
							case 3 <= len(expectation) && len(expectation) < 16:
								So(feed[0], assertions.ShouldResemble, expectation[0])
								So(feed[1].Promoted, assertions.ShouldBeTrue)
								So(feed[2:], assertions.ShouldResemble, expectation[1:])
							case 16 <= len(expectation):
								So(feed[0], assertions.ShouldResemble, expectation[0])
								So(feed[1].Promoted, assertions.ShouldBeTrue)
								So(feed[2:15], assertions.ShouldResemble, expectation[1:14])
								So(feed[15].Promoted, assertions.ShouldBeTrue)
								So(feed[16:], assertions.ShouldResemble, expectation[14:])
							}
						}
						So(resp.StatusCode(), ShouldEqual, http.StatusOK)
					}
					score--
				}
			}
		})

		Convey("Promoted posts won't appear in the neighborhood to NSFW-posts", func() {
			var promoted []protocol.Post
			for i := 0; i < 10; i++ {
				post := protocol.Post{
					Author:    fmt.Sprintf("t2_%08x", i),
					Subreddit: fmt.Sprintf("subreddit %d", i),
					Title:     fmt.Sprintf("XXX title %d", i),
					Score:     123,
					Promoted:  true,
				}
				{
					resp, err := r.SetBody(&post).Post("http://localhost:8080/submit")
					So(err, ShouldBeNil)
					So(string(resp.Body()), ShouldBeEmpty)
					So(resp.StatusCode(), ShouldEqual, http.StatusOK)
				}
				promoted = append(promoted, post)
				{
					var feed []protocol.Post
					resp, err := r.SetResult(&feed).Get("http://localhost:8080/feed")
					So(err, ShouldBeNil)
					So(feed, ShouldBeEmpty)
					So(resp.StatusCode(), ShouldEqual, http.StatusOK)
				}
			}

			score := 999
			var posts []protocol.Post
			for pages := 1; pages <= 10; pages++ {
				for i := 0; i < pageSize; i++ {
					post := protocol.Post{
						Author:    fmt.Sprintf("t2_%08x", score),
						Subreddit: fmt.Sprintf("subreddit %d", score),
						Title:     fmt.Sprintf("title %d", score),
						Score:     score,
						NSFW:      true,
					}
					{
						resp, err := r.SetBody(&post).Post("http://localhost:8080/submit")
						So(err, ShouldBeNil)
						So(string(resp.Body()), ShouldBeEmpty)
						So(resp.StatusCode(), ShouldEqual, http.StatusOK)
					}
					posts = append(posts, post)
					for p := 0; p < pages; p++ {
						var feed []protocol.Post
						resp, err := r.SetResult(&feed).SetQueryParam("page", strconv.Itoa(p)).Get("http://localhost:8080/feed")
						So(err, ShouldBeNil)
						{
							begin, end := p*pageSize, (p+1)*pageSize
							if end > len(posts) {
								end = len(posts)
							}
							expectation := posts[begin:end]
							So(feed, assertions.ShouldResemble, expectation)
						}
						So(resp.StatusCode(), ShouldEqual, http.StatusOK)
					}
					score--
				}
			}
		})

		// Reset(func() {
		// 	{
		// 		err := redisClient.Del(ctx, "feed").Err()
		// 		So(err, ShouldBeNil)
		// 	}
		// 	{
		// 		err := redisClient.Del(ctx, "promotion").Err()
		// 		So(err, ShouldBeNil)
		// 	}
		// 	{
		// 		err := redisClient.XTrim(ctx, "posts", 0).Err()
		// 		So(err, ShouldBeNil)
		// 	}
		// 	{
		// 		err := redisClient.Close()
		// 		So(err, ShouldBeNil)
		// 	}
		// })
	})
}
