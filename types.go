package dnsserver

import (
	"bytes"
	"encoding/binary"
	"errors"
	"strings"
)

type Header struct {
	ID              uint16
	Flags           uint16 // query/response, opcode, authoritative, truncated, recursion desired, recursion available, reserved, response code
	QuestionsCount  uint16
	AnswerCount     uint16
	AuthorityCount  uint16
	AdditionalCount uint16
}

func NewHeader(id, flags, questionCount, answerCount, authorityCount, additionalCount uint16) Header {
	return Header{
		ID:              id,
		Flags:           flags,
		QuestionsCount:  questionCount,
		AnswerCount:     answerCount,
		AuthorityCount:  authorityCount,
		AdditionalCount: additionalCount,
	}
}

// SetQuery sets the QR (Query/Response) bit in the DNS header flags.
// If isQuery = true, the message is a query (QR=0).
// If isQuery = false, the message is a response (QR=1).
func (h *Header) SetQuery(isQuery bool) {
	const qrMask uint16 = 1 << 15 // bit 15 is the QR bit
	if isQuery {
		// Clear the QR bit to indicate a query
		// AND with the inverse of the mask (1111_1111_1111_1111 ^ 1000_0000_0000_0000)
		h.Flags &^= qrMask
	} else {
		// Set the QR bit to indicate a response
		// OR with the mask (sets bit 15 to 1)
		h.Flags |= qrMask
	}
}

var (
	RCODE_NO_ERROR        = uint8(0)
	RCODE_FORMAT_ERROR    = uint8(1)
	RCODE_SERVER_FAILURE  = uint8(2)
	RCODE_NAME_ERROR      = uint8(3)
	RCODE_NOT_IMPLEMENTED = uint8(4)
	RCODE_REFUSED         = uint8(5)
)

// SetResponseCode sets the RCODE (Response Code) in the DNS header.
// RCODE occupies the lowest 4 bits of the Flags field.
func (h *Header) SetResponseCode(code uint8) {
	// Mask to keep only the lowest 4 bits of code
	code &= 0x0F
	// Clear the existing RCODE (lowest 4 bits)
	h.Flags &^= 0x000F
	// Set the new RCODE
	h.Flags |= uint16(code)
}

// The Header struct has no padding at the moment, so this can parse without relying on that.
// Future changes need to be aware of that.
func (h Header) MarshalBinary() ([]byte, error) {
	buf := bytes.NewBuffer(make([]byte, 0, 12))
	err := binary.Write(buf, binary.BigEndian, h)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (h *Header) UnmarshalBinary(data []byte) error {
	return binary.Read(bytes.NewReader(data), binary.BigEndian, h)
}

func NewHeaderFromBytes(data []byte) (Header, error) {
	header := Header{}
	err := header.UnmarshalBinary(data)
	if err != nil {
		return Header{}, err
	}
	return header, nil
}

type Question struct {
	Name  string
	Type  uint16
	Class uint16
}

func nameToLabels(name string) []string {
	return strings.Split(name, ".")
}

func (q Question) MarshalBinary() ([]byte, error) {
	buf := bytes.NewBuffer(make([]byte, 0, 12))
	for _, label := range nameToLabels(q.Name) {
		buf.WriteByte(byte(len(label)))
		buf.WriteString(label)
	}
	buf.WriteByte(0)
	binary.Write(buf, binary.BigEndian, q.Type)
	binary.Write(buf, binary.BigEndian, q.Class)

	return buf.Bytes(), nil
}

func NewQuestionFromBytes(data []byte) (Question, int, error) {
	if len(data) <= 0 {
		return Question{}, 0, errors.New("not enough data")
	}
	if data[0] <= 0 {
		return Question{}, 0, errors.New("invalid question")
	}

	var labels []string
	offset := 0

	for {
		length := int(data[offset])
		offset++
		if length == 0 {
			break
		}

		labels = append(labels, string(data[offset:offset+length]))
		offset += length
	}

	question := Question{
		Name:  strings.Join(labels, "."),
		Type:  binary.BigEndian.Uint16(data[offset : offset+2]),
		Class: binary.BigEndian.Uint16(data[offset+2 : offset+4]),
	}

	return question, offset + 4, nil
}

type Answer struct {
	Name   string
	Type   uint16
	Class  uint16
	TTL    uint32
	Length uint16
	Data   []byte
}

func (a Answer) MarshalBinary() ([]byte, error) {
	buf := bytes.NewBuffer(make([]byte, 0, 10))
	for _, label := range nameToLabels(a.Name) {
		buf.WriteByte(byte(len(label)))
		buf.WriteString(label)
	}
	buf.WriteByte(0)
	binary.Write(buf, binary.BigEndian, a.Type)
	binary.Write(buf, binary.BigEndian, a.Class)
	binary.Write(buf, binary.BigEndian, a.TTL)
	binary.Write(buf, binary.BigEndian, a.Length)
	buf.Write(a.Data)

	return buf.Bytes(), nil
}

type Message struct {
	Header    Header
	Questions []Question
	Answers   []Answer
}

func NewMessageFromBytes(data []byte) (Message, error) {
	h, err := NewHeaderFromBytes(data)
	if err != nil {
		return Message{}, err
	}

	questions := make([]Question, 0)
	for i := 0; i < int(h.QuestionsCount); i++ {
		q, n, err := NewQuestionFromBytes(data[12:])
		if err != nil {
			return Message{}, err
		}
		questions = append(questions, q)
		data = data[n:]
	}

	m := Message{
		Header:    h,
		Questions: questions,
	}

	m.Header.QuestionsCount = uint16(len(questions))
	return m, nil
}

func (m Message) MarshalBinary() ([]byte, error) {
	headerBytes, err := m.Header.MarshalBinary()
	if err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer(make([]byte, 0, 100))
	buf.Write(headerBytes)

	for _, q := range m.Questions {
		questionBytes, err := q.MarshalBinary()
		if err != nil {
			return nil, err
		}
		buf.Write(questionBytes)
	}

	for _, answer := range m.Answers {
		answerBytes, err := answer.MarshalBinary()
		if err != nil {
			return nil, err
		}
		buf.Write(answerBytes)
	}

	return buf.Bytes(), nil
}

func (m *Message) ProcessQuestions() {
	answers := make([]Answer, 0)
	for _, question := range m.Questions {
		// TODO: this is kinda mocked, but later should have real logic
		a := Answer{
			Name:  question.Name,
			Type:  question.Type,
			Class: question.Class,
			TTL:   60,                 // default value of TTL in seconds
			Data:  []byte{8, 8, 8, 8}, // mocked data
		}
		a.Length = uint16(len(a.Data))
		answers = append(answers, a)
	}

	m.AddAnswers(answers)
	m.SetResponse(len(answers))
}

func (m *Message) SetResponse(lenAnswers int) {
	m.Header.SetQuery(false)
	m.Header.AnswerCount = uint16(lenAnswers)

	// clear additional count because we don’t support EDNS; avoids malformed packet warnings in clients like dig
	m.Header.AdditionalCount = 0
}

func (m *Message) AddAnswers(answers []Answer) {
	m.Answers = answers
}
