package main

import (
	"blogaggregator/internal/config"
	"blogaggregator/internal/database"
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
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

	if len(os.Args) < 2 {
		log.Fatal("Error: No command given")
	}

	nextCommand := command{
		Name: os.Args[1],
		Args: os.Args[2:],
	}

	err = mainCommands.run(&mainState, nextCommand)
	if err != nil {
		log.Fatalf("Error running command %q: %v", nextCommand.Name, err)
	}
}

func handlerLogin(s *state, cmd command) error {
	if len(cmd.Args) < 1 {
		return fmt.Errorf("Login requires a username.\n")
	}

	username := cmd.Args[0]
	_, err := s.DB.GetUser(context.Background(), username)
	if err != nil {
		return fmt.Errorf("Cannot login with unknown user %q.\n", username)
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
		return fmt.Errorf("Register requires a username.\n")
	}

	username := cmd.Args[0]
	user, err := s.DB.CreateUser(context.Background(), database.CreateUserParams{
		ID:        uuid.New(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		Name:      username,
	})

	if err != nil {
		return fmt.Errorf("Error creating user: %v\n", err)
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
