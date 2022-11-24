package gostrreplacer

import (
	"bytes"
	"errors"
	"unicode/utf8"

	"golang.org/x/text/transform"
)

var ErrInvalidUTF8 = errors.New("invalid UTF-8 character")

func NewTransformer(matchStr, replaceStr string) transform.Transformer {
	return &customTransformer{matchRunes: []rune(matchStr), replaceRunes: []rune(replaceStr)}
}

type customTransformer struct {
	transform.NopResetter

	matchRunes   []rune
	replaceRunes []rune
}

func (t *customTransformer) Transform(dst, src []byte, atEOF bool) (int, int, error) {
	_src := src

	if len(_src) == 0 && atEOF {
		// srcが空 かつ EOFの場合は終了
		return 0, 0, nil
	}
	if !utf8.Valid(_src) {
		// utf8ではない(文字列ではない場合はError)
		return 0, 0, ErrInvalidUTF8
	}

	var nDst, nSrc int
	b := newCustomTransformerBuffer(t.matchRunes, t.replaceRunes)
	// srcが空になるまでloop
	for len(_src) > 0 {
		// 1 rune毎に処理する
		_, n := utf8.DecodeRune(_src)
		oneRune := _src[:n]

		buf, readBytes, ok := b.CanWrite(oneRune, len(_src) == n)
		if !ok {
			_src = _src[n:]
			continue
		}
		if nDst+len(buf) > len(dst) {
			// 書き込もうとしているBufferがdst buffer size超えてしまった場合は書き込まずにErrShortDstを返し次の処理に持ち越し
			return nDst, nSrc, transform.ErrShortDst
		}
		dstN := copy(dst[nDst:], buf)
		if dstN <= 0 {
			break
		}
		nSrc += readBytes
		nDst += dstN
		_src = _src[n:]
	}
	return nDst, nSrc, nil
}

func (t *customTransformer) replaceBytes() int {
	if len(t.replaceRunes) == 0 {
		return 0
	}
	return len([]byte(string(t.replaceRunes)))
}

type customTransformerBuffer struct {
	matchRunes   []rune
	replaceRunes []rune

	buffer []rune
	idx    int
}

func newCustomTransformerBuffer(matchRunes, replaceRunes []rune) customTransformerBuffer {
	return customTransformerBuffer{
		matchRunes: matchRunes, replaceRunes: replaceRunes,
		buffer: make([]rune, 0, len(matchRunes)),
	}
}

func (b *customTransformerBuffer) CanWrite(rb []byte, isLastRune bool) ([]byte, int, bool) {
	// Target文字列と等しいbyte列が来た場合bufferに格納する
	if b.isReplaceRune(rb, b.idx, isLastRune) {
		b.buffer = append(b.buffer, []rune(string(rb))...)
		b.idx++
		// Target文字列と全て等しい場合 replaceRunes を書き込むようにする
		if len(b.buffer) == len(b.matchRunes) {
			buf := b.buffer
			b.buffer = make([]rune, 0, len(b.matchRunes))
			b.idx = 0
			return []byte(string(b.replaceRunes)), len([]byte(string(buf))), true
		}
		return nil, len([]byte(string(b.buffer))), false
	}
	if len(b.buffer) > 0 {
		// prefixだけ一致していた異なる文字列の場合いままでbufferに格納していたbyte列を返却する
		buf := make([]byte, 0, len([]byte(string(b.buffer)))+len(rb))
		buf = append(buf, []byte(string(b.buffer))...)
		b.buffer = make([]rune, 0, len(b.matchRunes))
		b.idx = 0

		if b.isReplaceRune(rb, 0, isLastRune) {
			// 新規runeがTarget文字列の先頭とMatchする場合はBufferに書き込む
			b.buffer = append(b.buffer, []rune(string(rb))...)
			b.idx++
		} else {
			// 今回受け取ったbyteも返却
			buf = append(buf, rb...)
		}
		return buf, len(buf), true
	}
	return rb, len(rb), true
}

func (b *customTransformerBuffer) isReplaceRune(rb []byte, idx int, isLastRune bool) bool {
	if len(b.matchRunes) == 0 {
		return false
	}
	if len(b.matchRunes) <= idx {
		return false
	}
	if bytes.Equal([]byte(string(b.matchRunes[idx])), rb) {
		// Transformで処理する最後の文字でTarget文字列が揃いきらない場合は書き込む必要があるので除外
		if isLastRune && idx+1 < len(b.matchRunes) {
			return false
		}
		return true
	}
	return false
}
