package main

import (
    "bufio"
    "fmt"
    "io"
    "io/ioutil"
    "net/http"
    "os"
    "path/filepath"
    "strings"
    "time"
    "bytes"
    "mime/multipart"
)

func sendFileToTelegram(apiToken string, privateChannelID string, filePath string) {
    // Create a new multipart form
    requestBody := &bytes.Buffer{}
    writer := multipart.NewWriter(requestBody)

    // Add the file to the multipart form
    file, err := os.Open(filePath)
    if err != nil {
        fmt.Println("[-] Error opening file:", err)
        return
    }
    defer file.Close()
    part, err := writer.CreateFormFile("document", filepath.Base(filePath))
    if err != nil {
        fmt.Println("[-] Error creating form file:", err)
        return
    }

    // Create a TeeReader to write to both the part and a new buffer
    var buf bytes.Buffer
    teeReader := io.TeeReader(file, &buf)

    // Copy the TeeReader to the part
    fileSize, err := io.Copy(part, teeReader)
    if err != nil {
        fmt.Println("[-] Error copying file to form:", err)
        return
    }

    // Calculate the percentage of bytes written to the multipart form
    var percent float64
    for {
        // Check the size of the buffer
        bufSize := buf.Len()

        // Calculate the percentage of bytes written
        percent = float64(bufSize) / float64(fileSize) * 100

        // Print the percentage
        fmt.Printf("[*] Uploading: %s\n", filePath)

        // Wait for a short period of time before checking again
        time.Sleep(500 * time.Millisecond)

        // If the buffer is full, break out of the loop
        if bufSize == int(fileSize) {
            break
        }
    }

    // Add the chat ID and caption to the multipart form
    writer.WriteField("chat_id", privateChannelID)
    writer.WriteField("caption", fmt.Sprintf("[*] File size: %d bytes (%.2f%%)", fileSize, percent))

    // Close the multipart form
    err = writer.Close()
    if err != nil {
        fmt.Println("[-] Error closing multipart form:", err)
        return
    }

    // Send the multipart form to the Telegram API
    url := fmt.Sprintf("https://api.telegram.org/bot%s/sendDocument", apiToken)
    request, err := http.NewRequest("POST", url, requestBody)
    if err != nil {
        fmt.Println("[-] Error creating HTTP request:", err)
        return
    }
    request.Header.Set("Content-Type", writer.FormDataContentType())
    client := &http.Client{}
    response, err := client.Do(request)
    if err != nil {
        fmt.Println("[-] Error sending HTTP request:", err)
        return
    }
    defer response.Body.Close()

    // Check if the request was successful
    if response.StatusCode != 200 {
        fmt.Printf("[-] Error sending file to Telegram: %s\n", response.Status)
    } else {
        fmt.Println("[+] File sent to Telegram successfully!")
    }
}

func main() {
    // Check if the config file exists
    if len(os.Args) != 2 {
        fmt.Println("Usage: teledrop <file_path>")
        return
    }
    homeDir, err := os.UserHomeDir()
    if err != nil {
        fmt.Println("[-] Error getting user home directory:", err)
        return
    }
    configFile := filepath.Join(homeDir, ".teledrop.conf")
    if _, err := os.Stat(configFile); os.IsNotExist(err) {
        // If the config file does not exist, prompt the user for the API token and private channel ID
        fmt.Println("[?] The prompt will only show the first time if you still don't have config file.")
        fmt.Print("[+] Enter your Telegram bot API token: ")
        apiToken, err := readInput()
        if err != nil {
            fmt.Println("[-] Error reading input:", err)
            return
        }
        fmt.Print("[+] Enter your private channel ID: ")
        privateChannelID, err := readInput()
        if err != nil {
            fmt.Println("[-] Error reading input:", err)
            return
        }

        // Save the API token and private channel ID to the config file
        err = ioutil.WriteFile(configFile, []byte(fmt.Sprintf("API_TOKEN=%s\nPRIVATE_CHANNEL_ID=%s\n", apiToken, privateChannelID)), 0644)
        if err != nil {
            fmt.Println("[-] Error writing to config file:", err)
            return
        }
    }

    // Read the API token and private channel ID from the config file
    configData, err := ioutil.ReadFile(configFile)
    if err != nil {
        fmt.Println("[-] Error reading config file:", err)
        return
    }
    config := make(map[string]string)
    for _, line := range strings.Split(string(configData), "\n") {
        if line != "" {
            parts := strings.Split(line, "=")
            config[parts[0]] = parts[1]
        }
    }
    apiToken := config["API_TOKEN"]
    privateChannelID := config["PRIVATE_CHANNEL_ID"]

    filePath := os.Args[1]

    // Send the file to Telegram
    sendFileToTelegram(apiToken, privateChannelID, filePath)
}

func readInput() (string, error) {
    reader := bufio.NewReader(os.Stdin)
    input, err := reader.ReadString('\n')
    if err != nil {
        return "", err
    }
    return strings.TrimSpace(input), nil
}
