package smtp

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/slav123/email-catch/internal/config"
	tlsmanager "github.com/slav123/email-catch/internal/tls"
	"github.com/slav123/email-catch/pkg/email"
)

type Server struct {
	config            *config.Config
	listeners         []net.Listener
	processor         *email.Processor
	wg                sync.WaitGroup
	shutdown          chan struct{}
	letsencryptMgr    *tlsmanager.LetsEncryptManager
	renewalCtx        context.Context
	renewalCancel     context.CancelFunc
}

type Session struct {
	conn       net.Conn
	reader     *bufio.Reader
	writer     *bufio.Writer
	server     *Server
	helo       string
	mailFrom   string
	rcptTo     []string
	data       []byte
	tlsEnabled bool
}

func NewServer(cfg *config.Config, processor *email.Processor) *Server {
	renewalCtx, renewalCancel := context.WithCancel(context.Background())
	
	server := &Server{
		config:        cfg,
		processor:     processor,
		shutdown:      make(chan struct{}),
		renewalCtx:    renewalCtx,
		renewalCancel: renewalCancel,
	}

	if cfg.Server.TLS.LetsEncrypt.Enabled {
		leMgr, err := tlsmanager.NewLetsEncryptManager(tlsmanager.LetsEncryptConfig{
			Enabled:         cfg.Server.TLS.LetsEncrypt.Enabled,
			Domains:         cfg.Server.TLS.LetsEncrypt.Domains,
			Email:           cfg.Server.TLS.LetsEncrypt.Email,
			CacheDir:        cfg.Server.TLS.LetsEncrypt.CacheDir,
			Staging:         cfg.Server.TLS.LetsEncrypt.Staging,
			HTTPPort:        cfg.Server.TLS.LetsEncrypt.HTTPPort,
			RenewBefore:     cfg.Server.TLS.LetsEncrypt.RenewBeforeDays,
		})
		if err != nil {
			log.Printf("Failed to initialize Let's Encrypt manager: %v", err)
		} else {
			server.letsencryptMgr = leMgr
			log.Println("Let's Encrypt manager initialized successfully")
		}
	}

	return server
}

func (s *Server) Start() error {
	if s.letsencryptMgr != nil {
		if err := s.letsencryptMgr.ValidateDomains(); err != nil {
			return fmt.Errorf("Let's Encrypt domain validation failed: %w", err)
		}

		if err := s.letsencryptMgr.StartHTTPChallengeServer(); err != nil {
			log.Printf("Failed to start HTTP challenge server: %v", err)
		}

		certPath, keyPath, err := s.letsencryptMgr.GetCertificatePaths()
		if err != nil {
			log.Printf("Failed to get Let's Encrypt certificates: %v", err)
		} else {
			s.config.Server.TLS.CertFile = certPath
			s.config.Server.TLS.KeyFile = keyPath
			log.Printf("Using Let's Encrypt certificates: %s, %s", certPath, keyPath)
		}

		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			s.letsencryptMgr.StartAutoRenewal(s.renewalCtx)
		}()
	}

	for _, port := range s.config.Server.Ports {
		listener, err := s.startListener(port)
		if err != nil {
			s.Stop()
			return fmt.Errorf("failed to start listener on port %d: %w", port, err)
		}
		s.listeners = append(s.listeners, listener)
		
		s.wg.Add(1)
		go s.handleListener(listener, port)
		
		log.Printf("SMTP server listening on port %d", port)
	}
	
	return nil
}

func (s *Server) startListener(port int) (net.Listener, error) {
	addr := fmt.Sprintf("%s:%d", s.config.Server.Hostname, port)
	
	if s.config.Server.TLS.Enabled && (port == 465 || port == 993) {
		var tlsConfig *tls.Config
		
		if s.letsencryptMgr != nil {
			tlsConfig = s.letsencryptMgr.GetTLSConfig()
		} else {
			cert, err := tls.LoadX509KeyPair(s.config.Server.TLS.CertFile, s.config.Server.TLS.KeyFile)
			if err != nil {
				return nil, fmt.Errorf("failed to load TLS certificates: %w", err)
			}
			
			tlsConfig = &tls.Config{
				Certificates: []tls.Certificate{cert},
			}
		}
		
		return tls.Listen("tcp", addr, tlsConfig)
	}
	
	return net.Listen("tcp", addr)
}

func (s *Server) handleListener(listener net.Listener, port int) {
	defer s.wg.Done()
	
	for {
		select {
		case <-s.shutdown:
			return
		default:
			conn, err := listener.Accept()
			if err != nil {
				select {
				case <-s.shutdown:
					return
				default:
					log.Printf("Error accepting connection on port %d: %v", port, err)
					continue
				}
			}
			
			go s.handleConnection(conn, port)
		}
	}
}

func (s *Server) handleConnection(conn net.Conn, port int) {
	defer conn.Close()
	
	session := &Session{
		conn:   conn,
		reader: bufio.NewReader(conn),
		writer: bufio.NewWriter(conn),
		server: s,
		rcptTo: make([]string, 0),
	}
	
	log.Printf("New connection from %s on port %d", conn.RemoteAddr(), port)
	
	session.sendResponse(220, fmt.Sprintf("%s ESMTP Ready", s.config.Server.Hostname))
	
	for {
		line, err := session.reader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				log.Printf("Error reading from connection: %v", err)
			}
			break
		}
		
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		
		parts := strings.SplitN(line, " ", 2)
		command := strings.ToUpper(parts[0])
		args := ""
		if len(parts) > 1 {
			args = parts[1]
		}
		
		if !session.handleCommand(command, args) {
			break
		}
	}
}

func (s *Session) handleCommand(command, args string) bool {
	switch command {
	case "HELO":
		return s.handleHelo(args)
	case "EHLO":
		return s.handleEhlo(args)
	case "STARTTLS":
		return s.handleStartTLS()
	case "MAIL":
		return s.handleMail(args)
	case "RCPT":
		return s.handleRcpt(args)
	case "DATA":
		return s.handleData()
	case "RSET":
		return s.handleRset()
	case "QUIT":
		return s.handleQuit()
	case "NOOP":
		s.sendResponse(250, "OK")
		return true
	default:
		s.sendResponse(500, "Command not recognized")
		return true
	}
}

func (s *Session) handleHelo(args string) bool {
	if args == "" {
		s.sendResponse(501, "HELO requires domain address")
		return true
	}
	
	s.helo = args
	s.sendResponse(250, fmt.Sprintf("Hello %s", args))
	return true
}

func (s *Session) handleEhlo(args string) bool {
	if args == "" {
		s.sendResponse(501, "EHLO requires domain address")
		return true
	}
	
	s.helo = args
	
	// Build EHLO response as multi-line according to RFC 5321
	var responses []string
	responses = append(responses, fmt.Sprintf("%s Hello %s", s.server.config.Server.Hostname, args))
	
	if s.server.config.Server.TLS.Enabled && !s.tlsEnabled {
		responses = append(responses, "STARTTLS")
	}
	
	responses = append(responses, "SIZE 104857600")
	responses = append(responses, "8BITMIME")
	responses = append(responses, "PIPELINING")
	
	s.sendMultiLineResponse(250, responses)
	
	return true
}

func (s *Session) handleStartTLS() bool {
	if !s.server.config.Server.TLS.Enabled {
		s.sendResponse(502, "TLS not available")
		return true
	}
	
	if s.tlsEnabled {
		s.sendResponse(503, "TLS already active")
		return true
	}
	
	s.sendResponse(220, "Ready to start TLS")
	
	var tlsConfig *tls.Config
	
	if s.server.letsencryptMgr != nil {
		tlsConfig = s.server.letsencryptMgr.GetTLSConfig()
	} else {
		cert, err := tls.LoadX509KeyPair(s.server.config.Server.TLS.CertFile, s.server.config.Server.TLS.KeyFile)
		if err != nil {
			log.Printf("Failed to load TLS certificates: %v", err)
			s.sendResponse(454, "TLS not available")
			return true
		}
		
		tlsConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
		}
	}
	
	tlsConn := tls.Server(s.conn, tlsConfig)
	if err := tlsConn.Handshake(); err != nil {
		log.Printf("TLS handshake failed: %v", err)
		return false
	}
	
	s.conn = tlsConn
	s.reader = bufio.NewReader(tlsConn)
	s.writer = bufio.NewWriter(tlsConn)
	s.tlsEnabled = true
	
	s.helo = ""
	s.mailFrom = ""
	s.rcptTo = s.rcptTo[:0]
	
	return true
}

func (s *Session) handleMail(args string) bool {
	if s.helo == "" {
		s.sendResponse(503, "Need HELO first")
		return true
	}
	
	if !strings.HasPrefix(strings.ToUpper(args), "FROM:") {
		s.sendResponse(501, "Syntax error in MAIL command")
		return true
	}
	
	from := strings.TrimSpace(args[5:])
	if strings.HasPrefix(from, "<") && strings.HasSuffix(from, ">") {
		from = from[1 : len(from)-1]
	}
	
	s.mailFrom = from
	s.rcptTo = s.rcptTo[:0]
	s.sendResponse(250, "OK")
	return true
}

func (s *Session) handleRcpt(args string) bool {
	if s.mailFrom == "" {
		s.sendResponse(503, "Need MAIL first")
		return true
	}
	
	if !strings.HasPrefix(strings.ToUpper(args), "TO:") {
		s.sendResponse(501, "Syntax error in RCPT command")
		return true
	}
	
	to := strings.TrimSpace(args[3:])
	if strings.HasPrefix(to, "<") && strings.HasSuffix(to, ">") {
		to = to[1 : len(to)-1]
	}
	
	s.rcptTo = append(s.rcptTo, to)
	s.sendResponse(250, "OK")
	return true
}

func (s *Session) handleData() bool {
	if len(s.rcptTo) == 0 {
		s.sendResponse(503, "Need RCPT first")
		return true
	}
	
	s.sendResponse(354, "Start mail input; end with <CRLF>.<CRLF>")
	
	var data []byte
	for {
		line, err := s.reader.ReadBytes('\n')
		if err != nil {
			log.Printf("Error reading data: %v", err)
			return false
		}
		
		if len(line) >= 3 && line[0] == '.' && line[1] == '\r' && line[2] == '\n' {
			break
		}
		
		if len(line) > 0 && line[0] == '.' {
			line = line[1:]
		}
		
		data = append(data, line...)
	}
	
	s.data = data
	
	err := s.server.processor.ProcessEmail(s.mailFrom, s.rcptTo, data)
	if err != nil {
		log.Printf("Error processing email: %v", err)
		s.sendResponse(554, "Transaction failed")
		return true
	}
	
	s.sendResponse(250, "OK")
	
	s.mailFrom = ""
	s.rcptTo = s.rcptTo[:0]
	s.data = nil
	
	return true
}

func (s *Session) handleRset() bool {
	s.mailFrom = ""
	s.rcptTo = s.rcptTo[:0]
	s.data = nil
	s.sendResponse(250, "OK")
	return true
}

func (s *Session) handleQuit() bool {
	s.sendResponse(221, "Bye")
	return false
}

func (s *Session) sendResponse(code int, message string) {
	response := fmt.Sprintf("%d %s\r\n", code, message)
	s.writer.WriteString(response)
	s.writer.Flush()
}

func (s *Session) sendMultiLineResponse(code int, messages []string) {
	for i, message := range messages {
		if i == len(messages)-1 {
			// Last line uses space (final response)
			response := fmt.Sprintf("%d %s\r\n", code, message)
			s.writer.WriteString(response)
		} else {
			// Non-last lines use hyphen (continuation)
			response := fmt.Sprintf("%d-%s\r\n", code, message)
			s.writer.WriteString(response)
		}
	}
	s.writer.Flush()
}

func (s *Server) Stop() {
	close(s.shutdown)
	
	if s.renewalCancel != nil {
		s.renewalCancel()
	}
	
	for _, listener := range s.listeners {
		listener.Close()
	}
	
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		log.Println("SMTP server stopped gracefully")
	case <-time.After(30 * time.Second):
		log.Println("SMTP server stopped with timeout")
	}
}