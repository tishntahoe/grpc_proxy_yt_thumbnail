package main

import (
	"context"
	"flag"
	"fmt"
	"golang.org/x/net/html"
	"google.golang.org/grpc"
	pb "grpc_proxy_yt_thumbnail/grpc-proxy/proto/echo"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"regexp"
	"slices"

	db "grpc_proxy_yt_thumbnail/Services"
)

type arrayUrls []string

var urls arrayUrls

func (i *arrayUrls) String() string {
	return fmt.Sprintf("%v", *i)
}
func (i *arrayUrls) Set(v string) error {
	*i = append(*i, v)
	return nil
}

type sync_or_async_interface interface {
	Download() ([][]byte, string)
}

type server struct {
	pb.UnimplementedEchoServer
}

type mainstruct struct {
	s   *server
	ctx context.Context
	req *pb.Download
}
type async struct {
	*mainstruct
}
type sync struct {
	*mainstruct
}

func getMediaId(url string) string {
	reg := regexp.MustCompile("https://i\\.ytimg\\.com/vi/([^\"]*)/maxresdefault\\.jpg")
	res := reg.ReplaceAllString(url, "${1}")
	return res
}

func htmlThumbFinder(nd *html.Node) (response string, ok bool) {
	if nd.Type == html.ElementNode && nd.Data == "link" && nd.Attr[0].Val == "thumbnailUrl" {
		return nd.Attr[1].Val, true
	}
	for c := nd.FirstChild; c != nil; c = c.NextSibling {
		if response, ok = htmlThumbFinder(c); ok {
			return response, ok
		}
	}
	return "", false
}
func downloadFileToDirectory(thumbUrl string) *os.File {

	thumbnailPicUrl, _ := http.Get(thumbUrl)
	mediaId := getMediaId(thumbUrl)
	out, err := os.Create("downloadedFiles/" + mediaId + ".jpg")
	if err != nil {
		log.Fatal("Не удалось создать новый файл: ", err)
	}

	_, err = io.Copy(out, thumbnailPicUrl.Body)
	if err != nil {
		log.Fatal("Не удалось создать новый файл: ", err)
	}
	log.Println("Файл успешно скачан!")
	return out
}
func parseVidToThumb(url string) string {
	resp, err := http.Get(url)
	defer resp.Body.Close()
	if err != nil {
		log.Fatalf("Ссылка недействительна: %v", err)
	}
	parsed_resp, err := html.Parse(resp.Body)
	if err != nil {
		log.Fatalf("Не получилось распарсить код-html: %v", err)
	}
	url, ok := htmlThumbFinder(parsed_resp)
	if ok == false {
		log.Fatalf("Не нашел Thumbnail у этого видео")
	}
	return url
}
func convertToBytes(f *os.File) (bytesSlice [][]byte) {
	f.Seek(0, io.SeekStart)
	convertedImageToBytes, err := io.ReadAll(f)
	if err != nil {
		log.Fatalf("Ошибка на чтении файла: %v", err)
	}
	f.Close()
	bytesSlice = append(bytesSlice, convertedImageToBytes)
	return
}

func (s *server) PreDownload(ctx context.Context, req *pb.Download) (*pb.Response, error) {

	var downloadInterface sync_or_async_interface
	switch req.Async {
	case true:
		downloadInterface = async{&mainstruct{
			s,
			ctx,
			req,
		}}
	case false:
		downloadInterface = sync{&mainstruct{
			s,
			ctx,
			req,
		}}
	}
	output, thumbUrl := downloadInterface.Download() // сделать мапу
	dbProxy := db.DbConnectInfo.CreateConnectDb()
	for _, val := range output {
		db.InsertDb(dbProxy, db.Thumbnail_insrt{, val})
	}
	return &pb.Response{Resp: output}, nil
}

// Асинхронный метод (usage goroutines)
func (full async) Download() ([][]byte,string) {
	return nil,""
}

// Синхронный метод
func (full sync) Download() ([][]byte,string) {
	if len(full.req.Urls) != 1 {
		return nil, ""
	} // ИСПРАВИТЬ
	mainUrl := full.req.Urls[0]
	thumbUrl := parseVidToThumb(mainUrl)
	out := downloadFileToDirectory(thumbUrl)
	return convertToBytes(out), thumbUrl
}

func main() {
	mode := flag.Bool("async", false, "Async mode")
	flag.Var(&urls, "urls", "Some urls")
	flag.Parse() // Разбираем флаги

	strUrls := slices.Clone(urls)

	listener, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterEchoServer(s, &server{})
	log.Println("Старт grpc сервера")

	// открытие горутины для участия флагов и подключения утилиты
	go func() {
		conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
		if err != nil {
			log.Fatalf("Проблема с подключением ко второму серверу: %v", err)
		}
		defer conn.Close()

		client := pb.NewEchoClient(conn)
		if len(strUrls) != 0 {
			_, err = client.PreDownload(context.Background(), &pb.Download{Urls: strUrls, Async: *mode})
		}
		if err != nil {
			log.Fatalf("Ошибка вызова сервера обработки консольной утилиты: %v", err)
		}
		log.Printf("Сервер обработки консольной утилиты запущен")
	}()

	if err := s.Serve(listener); err != nil {
		log.Fatalf("Ошибка слушания листенера: %v", err)
	}

}
