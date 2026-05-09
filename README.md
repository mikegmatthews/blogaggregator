# Blog Aggregator
A simple CLI tool for following RSS feeds for multiple users.

This tool requires [PostgreSQL](https://www.postgresql.org/download/ "PostgreSQL download link") and [Go](https://go.dev/dl/ "Download link for the Go programming language") to be installed.

To install the tool, run:
`go install https://github.com/mikegmatthews/blogaggregator`

# Tool Usage
Once installed the application can be run with `gator <command> [options]`
## Commands
| Command   | Options                 | Description                                                                                                                                                                                                            |
|:----------|:------------------------|:-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| register  | username                | Registers a user with the given username with the tool. Registering a user automatically makes the newly registered user the current user.                                                                             |
| login     | username                | Sets the given user as the current user. The user must be registered to login.                                                                                                                                         |
| users     | <none>                  | Lists all registered users.                                                                                                                                                                                            |
| addfeed   | feed-name, feed-url     | Adds the given feed to the tool.  The feed is automatically added  the feeds followed by the current user.                                                                                                             |
| feeds     | <none>                  | Lists all feeds that have been added for all users.                                                                                                                                                                    |
| follow    | feed-url                | Assigns the given feed to be followed by the current user.                                                                                                                                                             |
| following | <none>                  | Lists all feeds that are followed by the current user.                                                                                                                                                                 |
| unfollow  | feed-url                | Unfollows the given feed for the current user.                                                                                                                                                                         |
| browse    | number-of-posts-to-show | Lists posts from all feeds followed by the current user, up to the given number of posts to show (this number defaults to 2 if none is provided), sorted by publish date starting with the most recently published.    |
| agg       | time-between-requests   | Aggregates posts from all feeds, separating feed requests by the given time.  The time can be provided in the form 00h00m00s (1h = one hour, 5m = 5 minutes, 30s = 30 seconds, 1m30s = 1 minute and 30 seconds, etc.). |

