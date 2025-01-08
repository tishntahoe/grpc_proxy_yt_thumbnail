package main

import (
	"context"
	"flag"
	"fmt"
	"golang.org/x/net/html"
	"google.golang.org/grpc"
	db "grpc_proxy_yt_thumbnail/Services"
	pb "grpc_proxy_yt_thumbnail/grpc-proxy/proto/echo"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"regexp"
	"slices"
	"sync"
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
	Download() map[string][]byte
}

type channelAsyncResult struct {
	name string
	val  []byte
}
type server struct {
	pb.UnimplementedEchoServer
}

type mainstruct struct {
	s   *server
	ctx context.Context
	req *pb.Download
}
type async_mode struct {
	*mainstruct
}
type sync_mode struct {
	*mainstruct
}

func getMediaId(url string) string {
	reg := regexp.MustCompile(`https://i\.ytimg\.com/vi/([^/]+)/[^?]*`)
	match := reg.FindStringSubmatch(url)
	return match[1] // Возвращаем первую группу захвата
	return ""       // Если совпадения не найдено
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
func downloadFileToDirectory(thumbUrl string) (f_out_slice *os.File) {
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
func parseVidToThumb(url string) (th_url string) {

	resp, err := http.Get(url)
	defer resp.Body.Close()
	if err != nil {
		log.Fatalf("Ссылка недействительна: %v", err)
	}
	parsed_resp, err := html.Parse(resp.Body)
	if err != nil {
		log.Fatalf("Не получилось распарсить код-html: %v", err)
	}
	th_url, ok := htmlThumbFinder(parsed_resp)
	if ok == false {
		log.Fatalf("Не нашел Thumbnail у этого видео")
	}

	return
}
func convertToBytes(f *os.File) (string, []byte) {
	f.Seek(0, io.SeekStart)
	convertedImageToBytes, err := io.ReadAll(f)
	if err != nil {
		log.Fatalf("Ошибка на чтении файла: %v", err)
	}
	f.Close()
	return f.Name(), convertedImageToBytes
}

func (s *server) PreDownload(ctx context.Context, req *pb.Download) (*pb.Response, error) {

	var downloadInterface sync_or_async_interface
	switch req.Async {
	case true:
		downloadInterface = async_mode{&mainstruct{
			s,
			ctx,
			req,
		}}
	case false:
		downloadInterface = sync_mode{&mainstruct{
			s,
			ctx,
			req,
		}}
	}

	dataMap := downloadInterface.Download() // сделать мапу
	dbProxy := db.DbConnectInfo.CreateConnectDb()
	matchedData := db.InsertDb_MatchData(dbProxy, dataMap)
	dbProxy.Close()
	log.Println("Закрыто")
	return &pb.Response{Resp: matchedData}, nil
}

// Асинхронный метод (usage goroutines)
func (full async_mode) Download() map[string][]byte {
	result := make(map[string][]byte)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, url := range full.req.Urls {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()

			thumbUrl := parseVidToThumb(url)
			out := downloadFileToDirectory(thumbUrl)
			log.Println("Начало конвертирования в байты")
			str, val := convertToBytes(out)
			log.Println("Конец конвертирования в байты")

			mu.Lock()
			result[str] = val
			mu.Unlock()
		}(url)
	}

	wg.Wait()
	return result
}

// Синхронный метод
func (full sync_mode) Download() map[string][]byte {
	result := make(map[string][]byte)
	if len(full.req.Urls) != 1 {
		return nil
	} // ИСПРАВИТЬ
	thumbUrl := parseVidToThumb(full.req.Urls[0])
	out := downloadFileToDirectory(thumbUrl)
	str, val := convertToBytes(out)
	result[str] = val
	return result
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

	if len(strUrls) != 0 {

		// открытие горутины для участия флагов и подключения утилиты
		go func() {
			conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
			if err != nil {
				log.Fatalf("Проблема с подключением ко второму серверу: %v", err)
			}
			defer conn.Close()

			client := pb.NewEchoClient(conn)
			if len(strUrls) != 0 {
				resp, _ := client.PreDownload(context.Background(), &pb.Download{Urls: strUrls, Async: *mode})
				fmt.Println(resp)
			}
			if err != nil {
				log.Fatalf("Ошибка вызова сервера обработки консольной утилиты: %v", err)
			}
			s.Stop()
		}()
	}
	if err := s.Serve(listener); err != nil {
		log.Fatalf("Ошибка слушания листенера: %v", err)
	}

}
