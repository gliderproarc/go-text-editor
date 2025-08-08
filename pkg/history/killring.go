package history

// KillRing is a minimal single-slot kill ring.
type KillRing struct {
    data string
}

// Set stores the killed text.
func (k *KillRing) Set(s string) { k.data = s }

// Get returns the last killed text.
func (k *KillRing) Get() string { return k.data }

// HasData reports whether the ring contains text.
func (k *KillRing) HasData() bool { return k.data != "" }

