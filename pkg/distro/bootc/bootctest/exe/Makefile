DESTDIR ?= .

.PHONY: all clean

all: $(DESTDIR)/bootc $(DESTDIR)/sleep

$(DESTDIR)/bootc: *.go
	mkdir -p $(DESTDIR)
	CGO_ENABLED=0 go build -o $@ $<

$(DESTDIR)/sleep: $(DESTDIR)/bootc
	cd $(DESTDIR) && ln -sf bootc sleep

clean:
	rm -f $(DESTDIR)/bootc $(DESTDIR)/sleep

