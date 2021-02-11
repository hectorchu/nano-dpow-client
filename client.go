package main

import (
	"errors"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type client struct {
	m      sync.Mutex
	conn   *websocket.Conn
	ch     map[uint]chan<- interface{}
	nextID uint
}

type response struct {
	ID          uint
	Work, Error string
}

func newClient() *client {
	return &client{ch: make(map[uint]chan<- interface{})}
}

func (c *client) connect() (err error) {
	if c.conn, _, err = websocket.DefaultDialer.Dial("wss://dpow.nanocenter.org/service_ws", nil); err != nil {
		return
	}
	go c.readLoop()
	return
}

func (c *client) request(hash, difficulty string, ch chan<- interface{}) (err error) {
	c.m.Lock()
	defer c.m.Unlock()
	c.nextID++
	c.ch[c.nextID] = ch
	if err = c.conn.WriteJSON(map[string]interface{}{
		"user":       *user,
		"api_key":    *apiKey,
		"id":         c.nextID,
		"hash":       hash,
		"difficulty": difficulty,
	}); err != nil {
		delete(c.ch, c.nextID)
	}
	return
}

func (c *client) readLoop() {
	for {
		var v response
		if err := c.conn.ReadJSON(&v); err != nil || v.Error != "" {
			if err == nil {
				err = errors.New(v.Error)
			}
			c.m.Lock()
			defer c.m.Unlock()
			for id, ch := range c.ch {
				ch <- err
				delete(c.ch, id)
			}
			c.conn.Close()
			for c.connect() != nil {
				time.Sleep(3 * time.Second)
			}
			return
		}
		c.m.Lock()
		ch := c.ch[v.ID]
		delete(c.ch, v.ID)
		c.m.Unlock()
		ch <- &v
	}
}
