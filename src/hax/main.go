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

	obj := MyObject{
		Name: "Bobby",
		Age:  31,
	}
	if err := db.PutObject([]byte("ObjectA"), &obj); err != nil {
		fmt.Fprintf(os.Stderr, "Couldn't write object: %v\n", err)
		return
	}
}

func readTest() {
	db, err := libdb.Open("ldbTest")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	var obj MyObject

	if err := db.GetObject([]byte("ObjectA"), &obj); err != nil {
		fmt.Fprintf(os.Stderr, "Couldn't read object: %v\n", err)
		return
	}

	fmt.Printf("Object: %v\n", obj)

	db.ForEach(func(key, value []byte) error {
		myObject := &MyObject{}
		if err := db.Decode(value, myObject); err != nil {
			return err
		}
		fmt.Printf("Enumerated object: %v\n", myObject)
		return nil
	})

}

func main() {
	writeTest()
	readTest()
}
