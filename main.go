package main

import (
	"blogaggregator/internal/config"
	"blogaggregator/internal/database"
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

type state struct {
	DB     *database.Queries
	Config *config.Config
}

type command struct {
	Name string
	Args []string
}

type commands struct {
	Commands map[string]func(*state, command) error
}

func (c *commands) run(s *state, cmd command) error {
	handler, ok := c.Commands[cmd.Name]
	if !ok {
		return fmt.Errorf("Unknown command: %q", cmd.Name)
	}

	return handler(s, cmd)
}

func (c *commands) register(name string, f func(*state, command) error) {
	c.Commands[name] = f
}

func main() {
	cfg, err := config.Read()
	if err != nil {
		log.Fatal(err)
	}

	mainState := state{
		Config: cfg,
	}

	db, err := sql.Open("postgres", cfg.DBUrl)
	if err != nil {
		log.Fatalf("Error: could not open database connection. %v", err)
	}
	dbQueries := database.New(db)
	mainState.DB = dbQueries

	mainCommands := commands{}
	mainCommands.Commands = make(map[string]func(*state, command) error)
	mainCommands.register("login", handlerLogin)
	mainCommands.register("register", handlerRegister)
	mainCommands.register("reset", handlerReset)
	mainCommands.register("users", handlerUsers)
	mainCommands.register("agg", handlerAgg)
	mainCommands.register("addfeed", middlewareLoggedIn(handlerAddFeed))
	mainCommands.register("feeds", handlerFeeds)
	mainCommands.register("follow", middlewareLoggedIn(handlerFollow))
	mainCommands.register("following", middlewareLoggedIn(handlerFollowing))
	mainCommands.register("unfollow", middlewareLoggedIn(handlerUnfollow))
	mainCommands.register("browse", middlewareLoggedIn(handlerBrowse))

	if len(os.Args) < 2 {
		log.Fatal("Error: No command given")
	}

	nextCommand := command{
		Name: os.Args[1],
		Args: os.Args[2:],
	}

	err = mainCommands.run(&mainState, nextCommand)
	if err != nil {
		log.Fatalf("Error running command %q: %v\n", nextCommand.Name, err)
	}
}

func handlerLogin(s *state, cmd command) error {
	if len(cmd.Args) < 1 {
		return fmt.Errorf("Login requires a username.")
	}

	username := cmd.Args[0]
	_, err := s.DB.GetUser(context.Background(), username)
	if err != nil {
		return fmt.Errorf("Cannot login with unknown user %q.", username)
	}

	err = s.Config.SetUser(username)
	if err != nil {
		return err
	}

	fmt.Println("User has been set")

	return nil
}

func handlerRegister(s *state, cmd command) error {
	if len(cmd.Args) < 1 {
		return fmt.Errorf("Register requires a username.")
	}

	username := cmd.Args[0]
	user, err := s.DB.CreateUser(context.Background(), database.CreateUserParams{
		ID:        uuid.New(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		Name:      username,
	})

	if err != nil {
		return fmt.Errorf("Error creating user: %v", err)
	}

	err = s.Config.SetUser(username)
	if err != nil {
		return err
	}

	fmt.Printf("User %q was created: %v\n", username, user)

	return nil
}

func handlerReset(s *state, _ command) error {
	return s.DB.ResetUsers(context.Background())
}

func handlerUsers(s *state, _ command) error {
	users, err := s.DB.GetUsers(context.Background())
	if err != nil {
		return nil
	}

	for _, user := range users {
		if s.Config.CurrentUser == user.Name {
			fmt.Printf("%s (current)\n", user.Name)
		} else {
			fmt.Println(user.Name)
		}
	}

	return nil
}

func handlerAgg(s *state, cmd command) error {
	if len(cmd.Args) < 1 {
		return fmt.Errorf("agg requires a time between requests (1s, 1m, 1h, etc.)")
	}
	timeBtwnReqs, err := time.ParseDuration(cmd.Args[0])
	if err != nil {
		return err
	}

	fmt.Printf("Collecting feeds every %s.\n", timeBtwnReqs)

	ticker := time.NewTicker(timeBtwnReqs)
	for ; ; <-ticker.C {
		scrapeFeeds(s)
	}
}

func middlewareLoggedIn(handler func(s *state, cmd command, user database.User) error) func(*state, command) error {
	return func(s *state, cmd command) error {
		currentUser, err := s.DB.GetUser(context.Background(), s.Config.CurrentUser)
		if err != nil {
			return err
		}

		return handler(s, cmd, currentUser)
	}
}

func handlerAddFeed(s *state, cmd command, currentUser database.User) error {
	if len(cmd.Args) < 2 {
		return fmt.Errorf("addFeed requires a name and URL.")
	}

	feedName := cmd.Args[0]
	feedUrl := cmd.Args[1]
	feed, err := s.DB.CreateFeed(context.Background(), database.CreateFeedParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      feedName,
		Url:       feedUrl,
		UserID:    currentUser.ID,
	})
	if err != nil {
		return err
	}

	_, err = s.DB.CreateFeedFollows(context.Background(), database.CreateFeedFollowsParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID:    currentUser.ID,
		FeedID:    feed.ID,
	})
	if err != nil {
		return err
	}

	fmt.Printf("%v\n", feed)

	return nil
}

func handlerFeeds(s *state, cmd command) error {
	feedsRow, err := s.DB.GetFeeds(context.Background())
	if err != nil {
		return err
	}

	for _, feed := range feedsRow {
		fmt.Printf("Feed %q:\n", feed.Name)
		fmt.Printf("  URL: %s\n", feed.Url)
		fmt.Printf("  Created by: %s\n", feed.UserName)
	}

	return nil
}

func handlerFollow(s *state, cmd command, currentUser database.User) error {
	if len(cmd.Args) < 1 {
		return fmt.Errorf("follow requires a URL")
	}

	feedUrl := cmd.Args[0]
	feed, err := s.DB.GetFeedByUrl(context.Background(), feedUrl)
	if err != nil {
		return err
	}

	feedFollow, err := s.DB.CreateFeedFollows(context.Background(), database.CreateFeedFollowsParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID:    currentUser.ID,
		FeedID:    feed.ID,
	})
	if err != nil {
		return err
	}

	fmt.Printf("Feed %q now followed by %s.", feedFollow.FeedName, feedFollow.UserName)

	return nil
}

func handlerFollowing(s *state, cmd command, currentUser database.User) error {
	allFeedFollows, err := s.DB.GetFeedFollowsForUser(context.Background(), currentUser.ID)
	if err != nil {
		return err
	}

	fmt.Printf("User %s is currently following:\n", currentUser.Name)
	for _, feed := range allFeedFollows {
		fmt.Println(feed.FeedName)
	}

	return nil
}

func handlerUnfollow(s *state, cmd command, currentUser database.User) error {
	if len(cmd.Args) < 1 {
		return fmt.Errorf("unfollow requires a feed URL")
	}

	feedUrl := cmd.Args[0]
	feed, err := s.DB.GetFeedByUrl(context.Background(), feedUrl)
	if err != nil {
		return err
	}

	err = s.DB.FeedUnfollow(context.Background(), database.FeedUnfollowParams{
		UserID: currentUser.ID,
		FeedID: feed.ID,
	})
	if err != nil {
		return err
	}

	return nil
}

func scrapeFeeds(s *state) error {
	nextFetch, err := s.DB.GetNextFeedToFetch(context.Background())
	if err != nil {
		return err
	}

	err = s.DB.MarkFeedFetched(context.Background(), nextFetch.ID)
	if err != nil {
		return err
	}

	rssFeeds, err := fetchFeed(context.Background(), nextFetch.Url)
	if err != nil {
		return err
	}

	fmt.Printf("%s Items:\n", rssFeeds.Channel.Title)
	for _, item := range rssFeeds.Channel.Item {
		pubDate, validErr := time.Parse("Mon, 02 Jan 2006 15:04:05 -0700", item.PubDate)

		_, err = s.DB.CreatePost(context.Background(), database.CreatePostParams{
			ID:        uuid.New(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Title:     item.Title,
			Url:       item.Link,
			Description: sql.NullString{
				String: item.Description,
				Valid:  item.Description != "",
			},
			PublishedAt: sql.NullTime{
				Time:  pubDate,
				Valid: validErr == nil,
			},
			FeedID: nextFetch.ID,
		})
		if err != nil && !strings.Contains(err.Error(), "duplicate key value") {
			fmt.Printf("Error creating entry for post from %s: %v\n", nextFetch.Name, err)
		}
	}

	return nil
}

func handlerBrowse(s *state, cmd command, currentUser database.User) error {
	var postLimit uint64 = 2
	if len(cmd.Args) > 0 {
		parsedLimit, err := strconv.ParseUint(cmd.Args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("Error parsing browse limit: %v", err)
		} else {
			postLimit = parsedLimit
		}
	}

	posts, err := s.DB.GetPostsForUser(context.Background(), database.GetPostsForUserParams{
		UserID: currentUser.ID,
		Limit:  int32(postLimit),
	})

	if err != nil {
		return err
	}

	for _, post := range posts {
		fmt.Println(post.Title)
		fmt.Println(post.Url)
		if post.PublishedAt.Valid {
			fmt.Printf("Published On: %s\n",
				post.PublishedAt.Time.Format("2 Jan 2006 15:04"))
		}
	}

	return nil
}
