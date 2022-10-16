package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

var root, err = os.Getwd()

const nosWorker = 20

type FileIndex struct {
	Name          string
	Link          string
	Folder        string
	Creation_date time.Time
	Modified_date time.Time
}

type fileData struct {
	index []FileIndex
	mutex sync.Mutex
}

func (f *fileData) Add(file ...FileIndex) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	f.index = append(f.index, file...)
}

type pathData struct {
	path  []string
	mutex sync.Mutex
}

func (p *pathData) Add(path ...string) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.path = append(p.path, path...)
}

func (p *pathData) Pop() string {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if len(p.path) > 0 {
		r := p.path[0]
		p.path = p.path[1:]
		return r
	} else {
		return ""
	}
}

func (p *pathData) Len() int {
	return len(p.path)
}

func index_folder(path string) ([]FileIndex, []string) {
	file_index_list := []FileIndex{}
	next := []string{}
	index, err := os.ReadDir(path)
	if err != nil {
		fmt.Println("Found error while reading path", path)
		return file_index_list, next
	}
	for _, d := range index {
		p := filepath.Join(path, d.Name())
		if d.IsDir() {
			next = append(next, p)
		} else {
			info, err := d.Info()
			if err != nil {
				fmt.Printf("prevent panic by handling failure accessing a path %q: %v\n", path, err)
				continue
			}
			s := info.Sys().(*syscall.Win32FileAttributeData)
			c := time.Unix(0, s.CreationTime.Nanoseconds())
			file_index_list = append(file_index_list, FileIndex{Name: d.Name(), Link: strings.ReplaceAll(strings.Replace(p, root, "", 1), "\\", "/") + "/", Folder: filepath.Base(path), Creation_date: c, Modified_date: info.ModTime()})
		}
	}

	return file_index_list, next
}

func worker(path *pathData, results *fileData, wg *sync.WaitGroup, id int) {
	defer wg.Done()
	p := path.Pop()
	if p == "" {
		return
	}
	fmt.Println("Worker id:", id, "Indexing...", p)
	index, n := index_folder(p)
	if len(index) > 0 {
		results.Add(index...)
	}
	if len(n) > 0 {
		path.Add(n...)
	}

}

func index_root() []FileIndex {
	path := pathData{path: []string{root}}
	results := fileData{index: []FileIndex{}}

	for path.Len() > 0 {
		wg := sync.WaitGroup{}
		wg.Add(nosWorker)
		for i := 0; i < nosWorker; i++ {
			go worker(&path, &results, &wg, i)
		}
		wg.Wait()
	}
	return results.index
}

func convert_to_JSON(index []FileIndex) string {
	b, err := json.Marshal(index)
	if err != nil {
		fmt.Println("error:", err)
	}
	return string(b)
}

func write_html(json string) {
	f, err := os.Create("search.html")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer f.Close()

	_, err2 := f.WriteString(fmt.Sprint(`<html>
	<head>
	<title>Search</title>
	<link href="https://unpkg.com/tabulator-tables@4.9.3/dist/css/tabulator.min.css" rel="stylesheet">
	
	</head>
	<body>
	
	<div id="example-table"></div>
	<p>Updated on 2022-10-08 11:06:06.388363</p>
	<script src="https://code.jquery.com/jquery-3.6.0.min.js" integrity="sha256-/xUj+3OJU5yExlq6GSYGSHk7tPXikynS7ogEvDej/m4=" crossorigin="anonymous"></script>
	<script type="text/javascript" src="https://unpkg.com/tabulator-tables@4.9.3/dist/js/tabulator.min.js"></script>
	<script>
	var tabledata =`, json, `;

	var table = new Tabulator("#example-table", {
		 data:tabledata,
		 layout:"fitColumns",      //fit columns to width of table
		responsiveLayout:"hide",  //hide columns that dont fit on the table
		tooltips:true,            //show tool tips on cells
		addRowPos:"top",          //when adding a new row, add it to the top of the table
		history:true,             //allow undo and redo actions on the table
		pagination:"local",       //paginate the data
		paginationSize:150,
		paginationSizeSelector:[150,250,500,1000,1500],         //allow 7 rows per page of data
		movableColumns:true,      //allow column order to be changed
		resizableRows:true,       //allow row order to be changed
		initialSort:[             //set the initial sort order of the data
			{column:"name", dir:"asc"},
				],
		 columns:[
			 {title:"Name", field:"Name",headerFilter:"input"},
			 {title:"Folder", field:"Folder",headerFilter:"input"},
			 {title:"Creation date", field:"Creation_date",headerFilter:"input"},
			 {title:"Modified date", field:"Modified_date",headerFilter:"input"},
		 ],
		 rowClick:function(e, row){ //trigger an alert message when the row is clicked
			 window.open(row.getData().Link,"_blank");}
	});
	</script>
	</body>`))

	if err2 != nil {
		fmt.Println(err2)
		return
	}

	fmt.Println("done")
}

func main() {
	index := index_root()
	json := convert_to_JSON(index)
	write_html(json)
}
