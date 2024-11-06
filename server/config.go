package main

import "time"

var CFGAdminAccount = map[string]string{
	"admin": "DinoKhoa@1234",
}
var CFGClientAccount = map[string]string{
	"client": "DinoKhoa1234",
	"admin": CFGAdminAccount["admin"],
}
var CFGMaxVNCConns = 100
var CFGMaxConcurrentOCR = 1
var CFGDb = "database.sqlite3"
var CFGClientPing = 5 * time.Second
var CFGClientTimeout = 60 * time.Second
var CFGIPInfoToken = "681af69a996680"
var CFGPasswords = []string{"123456", "password", "admin", "user", "default", "", "123456789", "111111", "password", "qwerty", "abc123", "12345678", "password1", "1234567", "123123", "pu", "god", "sex", "secret", "abc"}
var CFGVNCTimeout = "15"
var CFGVNCScreenshotBin = "./vncscreenshot"
var CFGTesseractBin = "tesseract"
