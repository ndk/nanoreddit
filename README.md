# nanoreddit

This service is to accept posts and represent them as a feed => Simplified version of the Reddit feed API that powers http://old.reddit.com. 

## API Reference
### POST /submit
Create new posts

Request
```
{
	"title": "title",
	"author": "t2_abcdefg9",
	"link": "https://reddit.com",
	"subreddit": "golang",
	"score": 999,
	"promoted": false,
	"nsfw": false
}
```
Constraints:
* author should be a random 8 lowercase letters or numbers prefixed with `t2_`
* a post cannot have both a link and content simultaneously

Example:
```
% curl -X POST --header "Content-Type: application/json" --data-raw '{"title":"title", "author":"t2_abcdefg9", "link":"https://reddit.com", "subreddit":"golang", "score":999, "promoted":false, "nsfw":false}' http://localhost:8080/submit
```
### GET /feed?page=0
Generate a paginated feed of posts

Response
```
[
  {
    "title": "title 1",
    "author": "t2_abcdefg9",
    "link": "https://reddit.com",
    "subreddit": "golang",
    "score": 999,
    "promoted": false,
    "nsfw": false
  },
  {
    "title": "enlarge something",
    "author": "t2_abcdefg9",
    "link": "https://reddit.com",
    "subreddit": "golang",
    "score": 999,
    "promoted": true,
    "nsfw": false
  },
  {
    "title": "title 2",
    "author": "t2_abcdefg9",
    "link": "https://reddit.com",
    "subreddit": "golang",
    "score": 99,
    "promoted": false,
    "nsfw": false
  }
]
```

Example:
```
% curl -X GET http://localhost:8080/feed?page=0
```

Constraints:
* It should be ranked by score, and the post with the highest score should show up first.
* It should be paginated, and each page should have at most 27 posts. Your API should
support fetching a specific page in the feed.
* If a page has 3 posts or greater, the second post should always be a promoted post if a
promoted post is available, regardless of the score.
* If a page has greater than 16 posts, the 16th post should always be a promoted post if a
promoted post is available, regardless of the score.
* As an exception to rules 3 and 4, a promoted post should never be shown adjacent
to an NSFW post. You can ignore rules 3 and 4 in this case.

## Components
```
           ______________                  ____________________
          |              | dump posts     |    Redis           |
*-/submit-| HTTP-server  |--------------->|                    |
          |              | process posts  |  posts(stream)     |
          | materializer |<---------------|  feed(sorted set)  |
          | |          ^ |  update feed   |  promoted(ring)    |
          | |          | |--------------->|                    |
          | v__________| |  update promo  |                    |
          |              |--------------->|                    |
          |              |merge feed&promo|                    |
*-/feed---|              |<---------------|____________________|
          |______________|
```

Service nanoreddit includes several routines:
1. http-server based on [hi](https://github.com/go-chi/chi). It accepts and validates requests. After this, all incoming posts go to the steam called `posts`. Of course, in production, it should be replaced something more reliable. For example, it can be Kafka.
2. The Materializer is a worker, which is processing posts from the stream `posts` and putting promoted and non-promoted posts into `promoted` and `feed` lists, respectively.
3. The feed is accessible by calling `/feed`. It reads `feed` from Redis, enriches with some promoted posts, and returns as a response.

## How to run

```
% docker-compose -f docker-compose.infra.yaml -f docker-compose.service.yaml up --build
```

To configure service you can use environment variables:
```
SERVICE_LISTEN_ADDRESS=:8080
SERVICE_SHUTDOWN_TIMEOUT=30s
SERVICE_LOGREQUESTS=true
FEED_PAGE_SIZE=25
ES_STREAM=posts
ES_FEED=feed
ES_PROMOTION=promotion
ES_GROUP=materializer
ES_CONSUMER=nanoreddit
REDIS_URL=redis://localhost:6379/0
LOGGER_LEVEL=info
LOGGER_TIMESTAMP=true
LOGGER_CALLER=true
LOGGER_PRETTY=true
```