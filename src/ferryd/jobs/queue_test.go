//
// Copyright © 2017 Ikey Doherty <ikey@solus-project.com>
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

package jobs

import (
	"testing"
)

func TestDequeueEmpty(t *testing.T) {
	q := NewQueue()
	_, err := q.Dequeue()
	if err != dequeueFail {
		t.Errorf("Dequeue should fail, expected: %s, found: %s", dequeueFail.Error(), err.Error())
	}
}

func TestFrontEmpty(t *testing.T) {
	q := NewQueue()
	_, err := q.Front()
	if err != frontFail {
		t.Errorf("Front should fail, expected: %s, found: %s", frontFail.Error(), err.Error())
	}
}

func TestEnqueueFront(t *testing.T) {
	q := NewQueue()
	j1 := JobEntry{1, BulkAdd}
	q.Enqueue(j1)
	j2, err := q.Front()
	if err != nil {
		t.Errorf("Did not expect error, found: %s", err.Error())
	}
	if j2.ID != 1 {
		t.Errorf("Job ID mismatch, expected: %d, found: %d", 1, j2.ID)
	}
	if j2.Type != BulkAdd {
		t.Errorf("Job Type mismatch, expected: %d, found: %d", BulkAdd, j2.Type)
	}
}

func TestEnqueueDequeue1(t *testing.T) {
	q := NewQueue()
	j1 := JobEntry{1, BulkAdd}
	q.Enqueue(j1)
	j2, err := q.Dequeue()
	if err != nil {
		t.Errorf("Did not expect error, found: %s", err.Error())
	}
	if j2.ID != 1 {
		t.Errorf("Job ID mismatch, expected: %d, found: %d", 1, j2.ID)
	}
	if j2.Type != BulkAdd {
		t.Errorf("Job Type mismatch, expected: %d, found: %d", BulkAdd, j2.Type)
	}
}

func TestEnqueueDequeue2(t *testing.T) {
	q := NewQueue()
	j1 := JobEntry{1, BulkAdd}
	j2 := JobEntry{3, CreateRepo}
	q.Enqueue(j1)
	q.Enqueue(j2)
	j3, err := q.Dequeue()
	if err != nil {
		t.Errorf("Did not expect error, found: %s", err.Error())
	}
	if j3.ID != j1.ID {
		t.Errorf("Job ID mismatch, expected: %d, found: %d", j1.ID, j3.ID)
	}
	if j3.Type != j1.Type {
		t.Errorf("Job Type mismatch, expected: %d, found: %d", j1.Type, j3.Type)
	}
	j4, err := q.Dequeue()
	if err != nil {
		t.Errorf("Did not expect error, found: %s", err.Error())
	}
	if j4.ID != j2.ID {
		t.Errorf("Job ID mismatch, expected: %d, found: %d", j1.ID, j2.ID)
	}
	if j4.Type != j2.Type {
		t.Errorf("Job Type mismatch, expected: %d, found: %d", j1.Type, j2.Type)
	}
}
