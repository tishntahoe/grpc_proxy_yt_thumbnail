package main

import (
	"context"
	"flag"
	"golang.org/x/net/html"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"regexp"

	"google.golang.org/grpc"
	pb "grpc_proxy_yt_thumbnail/grpc-proxy/proto/echo"
)

type server struct {
	pb.UnimplementedEchoServer
}

type sync_or_async_interface interface {
	Download() [][]byte
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
	output := downloadInterface.Download()

	log.Printf("Вывод: %s", output)
	return &pb.Response{Resp: output}, nil
}

// Асинхронный метод (usage goroutines)
func (full async) Download() [][]byte {
	return nil
}

// Синхронный метод
func (full sync) Download() [][]byte {
	if len(full.req.Urls) != 1 {
		return nil
	}
	mainUrl := full.req.Urls[0]
	resp, err := http.Get(mainUrl)
	defer resp.Body.Close()

	if err != nil {
		log.Fatalf("Ссылка недействительна: %v", err)
		return nil
	}
	parsed_resp, err := html.Parse(resp.Body)
	if err != nil {
		log.Fatalf("Не получилось распарсить код-html: %v", err)
	}

	url, ok := htmlThumbFinder(parsed_resp)
	if ok == false {
		log.Fatalf("Не нашел Thumbnail у этого видео")
	}

	thumbnailPicUrl, _ := http.Get(url)
	mediaId := getMediaId(url)

	os.Mkdir("downloadedFiles", 0777)
	out, err := os.Create("downloadedFiles/" + mediaId + ".jpg")
	if err != nil {
		log.Fatal("Не удалось создать новый файл: ", err)
	}
	defer out.Close()

	_, err = io.Copy(out, thumbnailPicUrl.Body)
	if err != nil {
		log.Fatal("Не удалось создать новый файл: ", err)
	}

	log.Println("Файл успешно скачан!")

	return nil
}

func main() {
	mode := flag.Bool("async", false, "Use Async")
	url := flag.String("url", "", "Some Urls")
	flag.Parse() // Разбираем флаги

	listener, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterEchoServer(s, &server{})
	log.Println("Старт")

	go func() {

		conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
		if err != nil {
			log.Fatalf("Проблема с подключением ко второму серверу: %v", err)
		}
		defer conn.Close()

		client := pb.NewEchoClient(conn)
		res, err := client.PreDownload(context.Background(), &pb.Download{Url: *url, Async *mode})
		if err != nil {
			log.Fatalf("Error calling Download: %v", err)
		}
		log.Printf("Ответ сервера: %s", res.Status)
	}()

	if err := s.Serve(listener); err != nil {
		log.Fatalf("Ошибка слушания листенера: %v", err)
	}

}
