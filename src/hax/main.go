//
// Copyright © 2017 Solus Project
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package main

import (
	"fmt"
	"libdb"
	"os"
)

// MyObject provided simply for serialisation tests
type MyObject struct {
	Name string
	Age  int
}

func writeTest() {
	db, err := libdb.Open("ldbTest")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	objA := MyObject{
		Name: "Bobby",
		Age:  31,
	}

	objB := MyObject{
		Name: "Johnny",
		Age:  26,
	}

	db.Update(func(d libdb.Database) error {
		if err := d.PutObject([]byte("ObjectA"), &objA); err != nil {
			fmt.Fprintf(os.Stderr, "Couldn't write object: %v\n", err)
			return err
		}
		if err := d.PutObject([]byte("ObjectB"), &objB); err != nil {
			fmt.Fprintf(os.Stderr, "Couldn't write object: %v\n", err)
			return err
		}
		return nil
		// return fmt.Errorf("nope no write")
	})

}

func readTest() {
	db, err := libdb.Open("ldbTest")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	db.View(func(r libdb.ReadOnlyView) error {
		return r.ForEach(func(key, value []byte) error {
			myObject := &MyObject{}
			if err := db.Decode(value, myObject); err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				return err
			}
			fmt.Printf("Enumerated object: %v, %v\n", string(key), myObject)
			return nil
		})
	})

	var obj MyObject

	if err := db.GetObject([]byte("ObjectA"), &obj); err != nil {
		fmt.Fprintf(os.Stderr, "Couldn't read object: %v\n", err)
		return
	}

	fmt.Printf("Object: %v\n", obj)
}

func main() {
	writeTest()
	readTest()
}
