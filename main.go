package main

import (
	"KevinGo/enhancedcontext"
	"KevinGo/ollama"
	"KevinGo/poll"
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/go-audio/audio"
	"github.com/go-audio/wav"
	"github.com/gordonklaus/portaudio"
	htgotts "github.com/hegedustibor/htgo-tts"
	"github.com/hegedustibor/htgo-tts/voices"
)

func main() {
	if _, err := os.Stat("assets"); os.IsNotExist(err) {
		os.Mkdir("assets", 0755)
	}

	fmt.Println("🔍 Verific dacă Ollama este disponibil...")
	if err := ollama.CheckOllamaStatus(); err != nil {
		fmt.Printf("❌ %v\n", err)
		fmt.Println("\n🔋 Pentru a instala și rula Ollama:")
		fmt.Println("1. Instalează: brew install ollama (sau https://ollama.ai/download)")
		fmt.Println("2. Rulează în terminal: ollama pull llama3.2")
		fmt.Println("3. Pornește serverul: ollama serve")
		fmt.Println("\n🛑 Aplicația se oprește...")
		return
	}
	fmt.Println("✅ Ollama este funcțional!")

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		fmt.Println("\n\n👋 Aplicația se închide gracefully...")
		portaudio.Terminate()
		os.Exit(0)
	}()

	portaudio.Initialize()
	defer portaudio.Terminate()

	fmt.Println("🎙️ Mod conversație continuă activat!")
	fmt.Println("🔢 Apasă Control+C pentru a ieși din aplicație")

	conversationCount := 0

	for {
		conversationCount++
		fmt.Printf("\n🗣️ Conversația #%d\n", conversationCount)
		fmt.Println("🎤 Apasă Enter pentru a porni înregistrarea...")
		bufio.NewReader(os.Stdin).ReadBytes('\n')

		fileWav := "assets/audio.wav"
		fileM4a := "assets/audio.m4a"

		in := make([]int16, 64)
		stream, err := portaudio.OpenDefaultStream(1, 0, 44100, len(in), in)
		if err != nil {
			log.Printf("❌ Eroare PortAudio: %v", err)
			continue
		}

		f, err := os.Create(fileWav)
		if err != nil {
			log.Printf("❌ Eroare creare fișier: %v", err)
			stream.Close()
			continue
		}
		enc := wav.NewEncoder(f, 44100, 16, 1, 1)

		fmt.Println("🎙 Înregistrare... Apasă Enter pentru a opri.")
		stream.Start()
		stopChan := make(chan bool)

		go func() {
			for {
				select {
				case <-stopChan:
					return
				default:
					stream.Read()
					buf := &audio.IntBuffer{
						Data:   intSlice(in),
						Format: &audio.Format{SampleRate: 44100, NumChannels: 1},
					}
					enc.Write(buf)
				}
			}
		}()

		bufio.NewReader(os.Stdin).ReadBytes('\n')
		stopChan <- true

		stream.Stop()
		stream.Close()
		enc.Close()
		f.Close()

		cmd := exec.Command("ffmpeg", "-y", "-i", fileWav, "-c:a", "aac", fileM4a)
		if err := cmd.Run(); err != nil {
			log.Printf("❌ Eroare la conversie: %v", err)
			continue
		}

		os.Remove(fileWav)

		fmt.Println("📝 Transcrierea audio...")
		transcribedText := poll.StartPolling()

		if transcribedText == "" {
			fmt.Println("❌ Nu s-a putut transcrie audio-ul")
			fmt.Println("🔄 Încearcă din nou...")
			continue
		}

		fmt.Printf("✅ Text transcris: %s\n", transcribedText)

		fmt.Println("🤖 Se procesează întrebarea cu Ollama...")
		fullContext := getKevinContext() + "\n\n" + enhancedcontext.GetSpecializedContext(transcribedText)
		response, err := ollama.AskWithContext(transcribedText, fullContext)
		if err != nil {
			log.Printf("❌ Eroare Ollama: %v", err)
			fmt.Println("🔄 Încearcă din nou...")
			continue
		}

		fmt.Printf("\n💬 Răspuns: %s\n", response)

		fmt.Println("🎵 Generez audio...")
		if err := generateTTSWithFallbacks(response); err != nil {
			log.Printf("❌ Nu s-a putut genera audio: %v", err)
			fmt.Println("🔄 Gata pentru următoarea întrebare...")
			continue
		}

		if err := playAudio(); err != nil {
			log.Printf("❌ Eroare la redarea audio: %v", err)
		}

		fmt.Println("🔄 Gata pentru următoarea întrebare...")
	}
}

func generateTTSWithFallbacks(response string) error {
	cleanAudioFolder()

	shortResponse := shortenResponse(response)
	fmt.Printf("📝 Text pentru TTS (%d caractere): %s\n", len(shortResponse), shortResponse)

	fallbacks := []struct {
		name string
		fn   func(string) error
	}{
		{"macOS say command", generateWithSayCommand},
		{"htgo-tts scurt", func(text string) error { return generateWithHTGOTTS(text, true) }},
		{"htgo-tts standard", func(text string) error { return generateWithHTGOTTS(text, false) }},
	}

	for _, fallback := range fallbacks {
		fmt.Printf("🔄 Încerc %s...\n", fallback.name)
		if err := fallback.fn(shortResponse); err != nil {
			fmt.Printf("❌ %s eșuat: %v\n", fallback.name, err)
			continue
		}

		if isValidAudioFile("assets/response.mp3") {
			fmt.Printf("✅ Audio generat cu succes folosind %s\n", fallback.name)
			return nil
		} else {
			fmt.Printf("❌ %s a generat fișier invalid\n", fallback.name)
		}
	}

	return fmt.Errorf("toate metodele TTS au eșuat")
}

func generateWithSayCommand(text string) error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("say command disponibil doar pe macOS")
	}

	if _, err := exec.LookPath("say"); err != nil {
		return fmt.Errorf("say command nu este disponibil")
	}

	tempFile := "assets/temp_response.aiff"
	finalFile := "assets/response.mp3"

	cmd := exec.Command("say", "-v", "Samantha", "-r", "180", "-o", tempFile, text)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("eroare say command: %w", err)
	}

	if _, err := exec.LookPath("ffmpeg"); err == nil {
		cmd = exec.Command("ffmpeg", "-y", "-i", tempFile, "-codec:a", "libmp3lame", "-b:a", "128k", finalFile)
		if err := cmd.Run(); err != nil {
			os.Remove(tempFile)
			return fmt.Errorf("eroare conversie ffmpeg: %w", err)
		}
		os.Remove(tempFile)
	} else {
		return os.Rename(tempFile, finalFile)
	}

	return nil
}

func generateWithHTGOTTS(text string, veryShort bool) error {
	if veryShort && len(text) > 100 {
		words := strings.Fields(text)
		if len(words) > 15 {
			text = strings.Join(words[:15], " ") + "."
		}
	}

	speech := htgotts.Speech{
		Folder:   "assets",
		Language: voices.English,
	}

	if err := speech.Speak(text); err != nil {
		return fmt.Errorf("eroare htgo-tts: %w", err)
	}

	time.Sleep(3 * time.Second)

	return findAndRenameGeneratedFile()
}

func findAndRenameGeneratedFile() error {
	files, err := filepath.Glob("assets/*.mp3")
	if err != nil {
		return err
	}

	for _, file := range files {
		if filepath.Base(file) != "response.mp3" {
			if isValidAudioFile(file) {
				return os.Rename(file, "assets/response.mp3")
			} else {
				os.Remove(file)
			}
		}
	}

	return fmt.Errorf("nu s-a găsit fișier audio valid")
}

func isValidAudioFile(filepath string) bool {
	file, err := os.Open(filepath)
	if err != nil {
		return false
	}
	defer file.Close()

	if info, err := file.Stat(); err != nil || info.Size() < 1000 {
		return false
	}

	header := make([]byte, 100)
	n, err := file.Read(header)
	if err != nil || n < 10 {
		return false
	}

	headerStr := string(header[:n])

	if strings.Contains(headerStr, "<!DOCTYPE") ||
		strings.Contains(headerStr, "<html") ||
		strings.Contains(headerStr, "<HTML") {
		return false
	}

	if len(header) >= 4 {
		if header[0] == 0xFF && (header[1]&0xFE) == 0xFA {
			return true
		}
		if string(header[:4]) == "FORM" {
			return true
		}
		if string(header[:4]) == "RIFF" {
			return true
		}
	}

	return true
}

func shortenResponse(response string) string {
	if strings.Contains(strings.ToLower(response), "weather") ||
		strings.Contains(strings.ToLower(response), "temperature") {

		sentences := strings.Split(response, ". ")
		if len(sentences) > 3 {
			return strings.Join(sentences[:3], ". ") + "."
		}
	}

	if len(response) > 250 {
		words := strings.Fields(response)
		result := ""
		for _, word := range words {
			if len(result)+len(word)+1 > 147 {
				break
			}
			if result != "" {
				result += " "
			}
			result += word
		}
		return result + "..."
	}

	return response
}

func cleanAudioFolder() {
	files, _ := filepath.Glob("assets/*.mp3")
	for _, file := range files {
		os.Remove(file)
	}
	files, _ = filepath.Glob("assets/*.aiff")
	for _, file := range files {
		os.Remove(file)
	}
}

func playAudio() error {
	audioFile := "assets/response.mp3"

	if _, err := os.Stat(audioFile); os.IsNotExist(err) {
		return fmt.Errorf("fișierul %s nu există", audioFile)
	}

	if !isValidAudioFile(audioFile) {
		return fmt.Errorf("fișierul audio nu este valid")
	}

	players := []struct {
		name string
		cmd  []string
	}{
		{"afplay", []string{"afplay", audioFile}},
		{"ffplay", []string{"ffplay", "-nodisp", "-autoexit", audioFile}},
		{"mpg123", []string{"mpg123", audioFile}},
		{"open", []string{"open", audioFile}}, // macOS default
	}

	for _, player := range players {
		if _, err := exec.LookPath(player.cmd[0]); err == nil {
			fmt.Printf("🔊 Redare cu %s...\n", player.name)
			cmd := exec.Command(player.cmd[0], player.cmd[1:]...)
			if err := cmd.Run(); err == nil {
				fmt.Printf("✅ Redare completă cu %s\n", player.name)
				return nil
			} else {
				fmt.Printf("❌ Eroare %s: %v\n", player.name, err)
			}
		}
	}

	return fmt.Errorf("nu s-a găsit niciun audio player funcțional")
}

func intSlice(in []int16) []int {
	out := make([]int, len(in))
	for i, v := range in {
		out[i] = int(v)
	}
	return out
}

func getKevinContext() string {
	return `Your name is Kira. You are a helpful AI assistant. 

Core Instructions:
- Always respond in English, regardless of the input language
- You are conversational and engaging
- KEEP RESPONSES CONCISE: maximum 30 words for weather queries, 50 words for other topics
- Be direct and to the point

Response Style Adaptation:
- If the user explicitly requests a specific tone (sarcastic, formal, funny, etc.), adopt that tone
- If the user asks you to "be like" someone or something, adapt accordingly while staying helpful
- If no specific style is mentioned, respond in a normal, friendly, and helpful manner

Response Guidelines:
- Be concise but thorough enough to be helpful
- Use natural, conversational English
- Stay respectful and appropriate regardless of requested style
- For weather: state temperature, conditions, and one clothing recommendation
- If asked technical questions, provide accurate information briefly
- If asked for creative content, be creative while staying within bounds

Remember: You are Kira, a helpful AI that adapts to what users need while always being respectful, concise, and speaking English.`
}
