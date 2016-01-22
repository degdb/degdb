default:
	go build -v .

cloc:
	find -not -iwholename '*.git*' -not -name '*.pb.go' -not -name '.' -not -name '~' -not -name '*.db' -not -name '*.key' -type f | xargs cloc

clean:
	rm ./degdb

.PHONY: clean default cloc
