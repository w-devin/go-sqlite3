// Copyright (C) 2023 Jonathan Giannuzzi <jonathan@giannuzzi.me>.
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

//go:build cgo && !libsqlite3
// +build cgo,!libsqlite3

package sqlite3

import (
	"database/sql"
	"os"
	"testing"
)

func TestCipher(t *testing.T) {
	ciphers := []string{
		"",
		"aes128cbc",
		"aes256cbc",
		"chacha20",
		"sqlcipher",
		"rc4",
	}
	keys := []string{
		// Passphrase with Key Derivation
		"passphrase",
		// Passphrase with Key Derivation starting with a digit
		"1passphrase",
		// Raw Key Data (Without Key Derivation)
		"x'2DD29CA851E7B56E4697B0E1F08507293D761A05CE4D1B628663F411A8086D99'",
		// Raw Key Data with Explicit Salt (Without Key Derivation)
		"x'98483C6EB40B6C31A448C22A66DED3B5E5E8D5119CAC8327B655C8B5C483648101010101010101010101010101010101'",
	}
	for _, cipher := range ciphers {
		for _, key := range keys {
			fname := TempFilename(t)
			uri := "file:" + fname + "?_key=" + key
			if cipher != "" {
				uri += "&_cipher=" + cipher
			}
			db, err := sql.Open("sqlite3", uri)
			if err != nil {
				os.Remove(fname)
				t.Errorf("sql.Open(\"sqlite3\", %q): %v", uri, err)
				continue
			}

			_, err = db.Exec("CREATE TABLE test (id int)")
			if err != nil {
				db.Close()
				os.Remove(fname)
				t.Errorf("failed creating test table for %q: %v", uri, err)
				continue
			}
			_, err = db.Exec("INSERT INTO test VALUES (1)")
			db.Close()
			if err != nil {
				os.Remove(fname)
				t.Errorf("failed inserting value into test table for %q: %v", uri, err)
				continue
			}

			db, err = sql.Open("sqlite3", "file:"+fname)
			if err != nil {
				os.Remove(fname)
				t.Errorf("sql.Open(\"sqlite3\", %q): %v", "file:"+fname, err)
				continue
			}
			_, err = db.Exec("SELECT id FROM test")
			db.Close()
			if err == nil {
				os.Remove(fname)
				t.Errorf("didn't expect to be able to access the encrypted database %q without a passphrase", fname)
				continue
			}

			badUri := "file:" + fname + "?_key=bogus"
			if cipher != "" {
				badUri += "&_cipher=" + cipher
			}
			db, err = sql.Open("sqlite3", badUri)
			if err != nil {
				os.Remove(fname)
				t.Errorf("sql.Open(\"sqlite3\", %q): %v", badUri, err)
				continue
			}
			_, err = db.Exec("SELECT id FROM test")
			db.Close()
			if err == nil {
				os.Remove(fname)
				t.Errorf("didn't expect to be able to access the encrypted database %q with a bogus passphrase", fname)
				continue
			}

			db, err = sql.Open("sqlite3", uri)
			if err != nil {
				os.Remove(fname)
				t.Errorf("sql.Open(\"sqlite3\", %q): %v", uri, err)
				continue
			}
			_, err = db.Exec("SELECT id FROM test")
			db.Close()
			os.Remove(fname)
			if err != nil {
				t.Errorf("unable to query test table for %q: %v", uri, err)
				continue
			}
		}
	}
}
