package dnsserver

import (
	"bytes"
	"encoding/binary"
	"errors"
	"strings"
)

type Header struct {
	ID              uint16
	Flags           uint16
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

func (q Question) MarshalBinary() ([]byte, error) {
	buf := bytes.NewBuffer(make([]byte, 0, 12))
	for label := range strings.SplitSeq(q.Name, ".") {
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

type Message struct {
	Header    Header
	Questions []Question
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

	return Message{
		Header:    h,
		Questions: questions,
	}, nil
}

func (m Message) MarshalBinary() ([]byte, error) {
	headerBytes, err := m.Header.MarshalBinary()
	if err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer(make([]byte, 0, 12))
	buf.Write(headerBytes)

	for _, question := range m.Questions {
		questionBytes, err := question.MarshalBinary()
		if err != nil {
			return nil, err
		}
		buf.Write(questionBytes)
	}

	return buf.Bytes(), nil
}
