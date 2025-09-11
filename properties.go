package mst

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
)

type PropsType int

const (
	PROP_TYPE_STRING = iota
	PROP_TYPE_INT
	PROP_TYPE_FLOAT
	PROP_TYPE_BOOL
	PROP_TYPE_ARRAY
	PROP_TYPE_MAP
)

type PropsValue struct {
	Type  PropsType
	Value interface{}
}

type Properties map[string]PropsValue

// PropertiesMarshal 序列化Properties
func PropertiesMarshal(wt io.Writer, props *Properties) error {
	// 嵌套函数：序列化单个PropsValue
	marshalPropsValue := func(wt io.Writer, value PropsValue) error {
		switch value.Type {
		case PROP_TYPE_STRING:
			str := value.Value.(string)
			if err := writeLittleUint32(wt, uint32(len(str))); err != nil {
				return fmt.Errorf("write string len failed: %w", err)
			}
			if _, err := wt.Write([]byte(str)); err != nil {
				return fmt.Errorf("write string content failed: %w", err)
			}
		case PROP_TYPE_INT:
			intVal := value.Value.(int64)
			if err := writeLittleInt64(wt, intVal); err != nil {
				return fmt.Errorf("write int64 failed: %w", err)
			}
		case PROP_TYPE_FLOAT:
			floatVal := value.Value.(float64)
			if err := writeLittleFloat64(wt, floatVal); err != nil {
				return fmt.Errorf("write float64 failed: %w", err)
			}
		case PROP_TYPE_BOOL:
			val := uint8(0)
			if value.Value.(bool) {
				val = 1
			}
			if err := writeLittleUint8(wt, val); err != nil {
				return fmt.Errorf("write bool failed: %w", err)
			}
		case PROP_TYPE_ARRAY:
			arr := value.Value.([]PropsValue)
			if err := writeLittleUint32(wt, uint32(len(arr))); err != nil {
				return fmt.Errorf("write array len failed: %w", err)
			}
			for _, item := range arr {
				if err := writeLittleUint32(wt, uint32(item.Type)); err != nil {
					return fmt.Errorf("write array item type failed: %w", err)
				}
				if err := marshalPropsValue(wt, item); err != nil {
					return fmt.Errorf("write array item failed: %w", err)
				}
			}
		case PROP_TYPE_MAP:
			subProps := value.Value.(Properties)
			if err := PropertiesMarshal(wt, &subProps); err != nil {
				return fmt.Errorf("write map properties failed: %w", err)
			}
		}
		return nil
	}

	if props == nil {
		if err := writeLittleUint32(wt, 0); err != nil {
			return fmt.Errorf("write nil marker failed: %w", err)
		}
		return nil
	}

	// 写入Properties数量
	propsCount := uint32(len(*props))
	if err := writeLittleUint32(wt, propsCount); err != nil {
		return fmt.Errorf("write properties count failed: %w", err)
	}

	for key, value := range *props {
		// 写入key长度
		keyLen := uint32(len(key))
		if err := writeLittleUint32(wt, keyLen); err != nil {
			return fmt.Errorf("write key len failed: %w", err)
		}
		// 写入key内容
		if _, err := wt.Write([]byte(key)); err != nil {
			return fmt.Errorf("write key content failed: %w", err)
		}

		// 写入类型
		if err := writeLittleUint32(wt, uint32(value.Type)); err != nil {
			return fmt.Errorf("write value type failed: %w", err)
		}

		// 根据类型写入值
		if err := marshalPropsValue(wt, value); err != nil {
			return fmt.Errorf("write value failed: %w", err)
		}
	}
	return nil
}

// PropertiesUnMarshal 反序列化Properties
func PropertiesUnMarshal(rd io.Reader) *Properties {
	// 读取Properties数量
	var size uint32
	if err := readLittleByte(rd, &size); err != nil {
		return nil
	}

	// 安全检查
	if size > 1000 { // 设置合理的上限
		return nil
	}

	props := make(Properties)
	for i := uint32(0); i < size; i++ {
		// 读取key长度
		var keyLen uint32
		if err := readLittleByte(rd, &keyLen); err != nil {
			return nil
		}

		// 安全检查
		if keyLen > 100 { // 设置合理的key长度上限
			return nil
		}

		// 读取key内容
		keyBytes := make([]byte, keyLen)
		if _, err := io.ReadFull(rd, keyBytes); err != nil {
			return nil
		}
		key := string(keyBytes)

		// 读取类型
		var propType uint32
		if err := readLittleByte(rd, &propType); err != nil {
			return nil
		}

		// 根据类型读取值
		value := unmarshalPropsValue(rd, PropsType(propType))
		if value.Type == -1 { // 表示反序列化失败
			return nil
		}

		// 类型验证
		if uint32(value.Type) != propType {
			return nil
		}

		props[key] = value
	}

	return &props
}

// 辅助函数，用于反序列化单个PropsValue
func unmarshalPropsValue(rd io.Reader, propType PropsType) PropsValue {
	var value interface{}
	var err error

	switch propType {
	case PROP_TYPE_STRING:
		var strLen uint32
		if err = readLittleByte(rd, &strLen); err != nil {
			return PropsValue{Type: -1}
		}
		// 添加安全检查
		if strLen > 100000 {
			return PropsValue{Type: -1}
		}
		strBytes := make([]byte, strLen)
		if _, err = io.ReadFull(rd, strBytes); err != nil {
			return PropsValue{Type: -1}
		}
		value = string(strBytes)
	case PROP_TYPE_INT:
		var intVal int64
		if err = readLittleByte(rd, &intVal); err != nil {
			return PropsValue{Type: -1}
		}
		value = intVal
	case PROP_TYPE_FLOAT:
		var floatVal float64
		if err = readLittleByte(rd, &floatVal); err != nil {
			return PropsValue{Type: -1}
		}
		value = floatVal
	case PROP_TYPE_BOOL:
		var boolVal uint8
		if err = readLittleByte(rd, &boolVal); err != nil {
			return PropsValue{Type: -1}
		}
		value = boolVal == 1
	case PROP_TYPE_ARRAY:
		var arrLen uint32
		if err = readLittleByte(rd, &arrLen); err != nil {
			return PropsValue{Type: -1}
		}
		// 添加安全检查
		if arrLen > 100000 {
			return PropsValue{Type: -1}
		}
		arr := make([]PropsValue, arrLen)
		for i := uint32(0); i < arrLen; i++ {
			var itemType uint32
			if err = readLittleByte(rd, &itemType); err != nil {
				return PropsValue{Type: -1}
			}
			item := unmarshalPropsValue(rd, PropsType(itemType))
			if item.Type == -1 {
				return PropsValue{Type: -1}
			}
			// 类型验证
			if uint32(item.Type) != itemType {
				return PropsValue{Type: -1}
			}
			arr[i] = item
		}
		value = arr
	case PROP_TYPE_MAP:
		subProps := PropertiesUnMarshal(rd)
		if subProps == nil {
			return PropsValue{Type: -1}
		}
		value = *subProps
	default:
		return PropsValue{Type: -1}
	}

	if err != nil {
		return PropsValue{Type: -1}
	}

	return PropsValue{Type: propType, Value: value}
}

// writeLittleUint32 写入小端序uint32
func writeLittleUint32(wt io.Writer, v uint32) error {
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, v)
	_, err := wt.Write(buf)
	return err
}

// writeLittleInt64 写入小端序int64
func writeLittleInt64(wt io.Writer, v int64) error {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, uint64(v))
	_, err := wt.Write(buf)
	return err
}

// writeLittleFloat64 写入小端序float64
func writeLittleFloat64(wt io.Writer, v float64) error {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, math.Float64bits(v))
	_, err := wt.Write(buf)
	return err
}

// writeLittleUint8 写入小端序uint8
func writeLittleUint8(wt io.Writer, v uint8) error {
	_, err := wt.Write([]byte{v})
	return err
}

// 辅助函数，用于序列化单个PropsValue
func marshalPropsValue(wt io.Writer, value PropsValue) error {
	switch value.Type {
	case PROP_TYPE_STRING:
		str := value.Value.(string)
		writeLittleByte(wt, uint32(len(str)))
		wt.Write([]byte(str))
	case PROP_TYPE_INT:
		writeLittleByte(wt, value.Value.(int64))
	case PROP_TYPE_FLOAT:
		writeLittleByte(wt, value.Value.(float64))
	case PROP_TYPE_BOOL:
		if value.Value.(bool) {
			writeLittleByte(wt, uint8(1))
		} else {
			writeLittleByte(wt, uint8(0))
		}
	case PROP_TYPE_ARRAY:
		arr := value.Value.([]PropsValue)
		writeLittleByte(wt, uint32(len(arr)))
		for _, item := range arr {
			writeLittleByte(wt, uint32(item.Type))
			marshalPropsValue(wt, item)
		}
	case PROP_TYPE_MAP:
		subProps := value.Value.(Properties)
		PropertiesMarshal(wt, &subProps)
	}
	return nil
}
