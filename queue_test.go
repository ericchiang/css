package css

import "testing"

func TestQueue(t *testing.T) {
	t1 := token{tokenDelim, "*", "*", 0, 0, ""}
	t2 := token{tokenIdent, "foo", "foo", 0, 0, ""}
	t3 := token{tokenIdent, "bar", "bar", 0, 0, ""}
	t4 := token{tokenIdent, "spam", "spam", 0, 0, ""}

	_, _ = t3, t4

	q := newQueue(2)
	q.push(t1)
	if got := q.get(0); got != t1 {
		t.Errorf("get(0) from queue with single element, got%#v, want=%#v", got, t1)
	}
	q.push(t2)
	if got := q.get(0); got != t1 {
		t.Errorf("get(0) from queue with two elements, got%#v, want=%#v", got, t1)
	}
	if got := q.get(1); got != t2 {
		t.Errorf("get(1) from queue with two elements, got%#v, want=%#v", got, t2)
	}

	if got := q.pop(); got != t1 {
		t.Errorf("pop() from queue with two elements, got%#v, want=%#v", got, t1)
	}
	q.push(t3)
	if got := q.get(0); got != t2 {
		t.Errorf("get(0) from queue with two elements after requeue, got%#v, want=%#v", got, t2)
	}
	if got := q.get(1); got != t3 {
		t.Errorf("get(1) from queue with two elements after requeue, got%#v, want=%#v", got, t3)
	}
	if got := q.pop(); got != t2 {
		t.Errorf("pop() from queue with single element, got%#v, want=%#v", got, t1)
	}
}
