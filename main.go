package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"net/http"
	"time"

	log "github.com/Sirupsen/logrus"
)

const timeout = 1 * time.Second

func main() {
	http.HandleFunc("/test", handleTest)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func genCtxID() (string, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%X", b), nil
}

func fromContext(ctx context.Context, key interface{}) (interface{}, error) {
	value, ok := ctx.Value(key).(interface{})
	if !ok {
		return nil, fmt.Errorf("Error retrieving %s from context", key)
	}
	return value, nil
}

func wait(ctx context.Context) error {
	ctxID, err := fromContext(ctx, "ctxid")
	if err != nil {
		return err
	}

	logger := log.WithFields(log.Fields{"ctxid": ctxID})

	// Toggle between 0 and 2 seconds (Recall: timeout is 1 second)
	var secondsToSleep int
	if time.Now().UnixNano()%2 == 0 {
		secondsToSleep = 0
	} else {
		secondsToSleep = 2
	}

	logger.Infof("Sleeping for %d seconds...", secondsToSleep)
	time.Sleep(time.Duration(secondsToSleep) * time.Second)

	return nil
}

func doSomething(ctx context.Context) error {
	ch := make(chan error)
	go func() {
		err := wait(ctx)
		ch <- err
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-ch:
		return err
	}
}

func handleTest(w http.ResponseWriter, req *http.Request) {
	rootCtx := context.Background()

	ctx, cancel := context.WithTimeout(rootCtx, timeout)
	defer cancel()

	ctxID, err := genCtxID()
	if err != nil {
		http.Error(w, fmt.Sprintf("%+v", err), http.StatusInternalServerError)
		return
	}

	ctx = context.WithValue(ctx, "ctxid", ctxID)
	logger := log.WithFields(log.Fields{"ctxid": ctxID})

	err = doSomething(ctx)
	if err != nil {
		logger.Errorf("%+v", err)
		http.Error(w, fmt.Sprintf("%+v", err), http.StatusRequestTimeout)
		return
	}

	fmt.Fprintf(w, fmt.Sprintf("Response within %d second(s)", timeout/time.Second))
}
