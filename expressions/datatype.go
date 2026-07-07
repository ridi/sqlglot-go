package expressions

type DType string

const (
	DTypeArray                   DType = "ARRAY"
	DTypeAggregateFunction       DType = "AGGREGATEFUNCTION"
	DTypeSimpleAggregateFunction DType = "SIMPLEAGGREGATEFUNCTION"
	DTypeBigDecimal              DType = "BIGDECIMAL"
	DTypeBigInt                  DType = "BIGINT"
	DTypeBigNum                  DType = "BIGNUM"
	DTypeBigSerial               DType = "BIGSERIAL"
	DTypeBinary                  DType = "BINARY"
	DTypeBit                     DType = "BIT"
	DTypeBlob                    DType = "BLOB"
	DTypeBoolean                 DType = "BOOLEAN"
	DTypeBpchar                  DType = "BPCHAR"
	DTypeChar                    DType = "CHAR"
	DTypeCharacterSet            DType = "CHARACTER_SET"
	DTypeDate                    DType = "DATE"
	DTypeDate32                  DType = "DATE32"
	DTypeDateMultiRange          DType = "DATEMULTIRANGE"
	DTypeDateRange               DType = "DATERANGE"
	DTypeDatetime                DType = "DATETIME"
	DTypeDatetime2               DType = "DATETIME2"
	DTypeDatetime64              DType = "DATETIME64"
	DTypeDecimal                 DType = "DECIMAL"
	DTypeDecimal32               DType = "DECIMAL32"
	DTypeDecimal64               DType = "DECIMAL64"
	DTypeDecimal128              DType = "DECIMAL128"
	DTypeDecimal256              DType = "DECIMAL256"
	DTypeDecFloat                DType = "DECFLOAT"
	DTypeDouble                  DType = "DOUBLE"
	DTypeDynamic                 DType = "DYNAMIC"
	DTypeEnum                    DType = "ENUM"
	DTypeEnum8                   DType = "ENUM8"
	DTypeEnum16                  DType = "ENUM16"
	DTypeFile                    DType = "FILE"
	DTypeFixedString             DType = "FIXEDSTRING"
	DTypeFloat                   DType = "FLOAT"
	DTypeGeography               DType = "GEOGRAPHY"
	DTypeGeographyPoint          DType = "GEOGRAPHYPOINT"
	DTypeGeometry                DType = "GEOMETRY"
	DTypePoint                   DType = "POINT"
	DTypeRing                    DType = "RING"
	DTypeLineString              DType = "LINESTRING"
	DTypeMultiLineString         DType = "MULTILINESTRING"
	DTypePolygon                 DType = "POLYGON"
	DTypeMultiPolygon            DType = "MULTIPOLYGON"
	DTypeHllSketch               DType = "HLLSKETCH"
	DTypeHstore                  DType = "HSTORE"
	DTypeImage                   DType = "IMAGE"
	DTypeInet                    DType = "INET"
	DTypeInt                     DType = "INT"
	DTypeInt128                  DType = "INT128"
	DTypeInt256                  DType = "INT256"
	DTypeInt4MultiRange          DType = "INT4MULTIRANGE"
	DTypeInt4Range               DType = "INT4RANGE"
	DTypeInt8MultiRange          DType = "INT8MULTIRANGE"
	DTypeInt8Range               DType = "INT8RANGE"
	DTypeInterval                DType = "INTERVAL"
	DTypeIpAddress               DType = "IPADDRESS"
	DTypeIpPrefix                DType = "IPPREFIX"
	DTypeIpv4                    DType = "IPV4"
	DTypeIpv6                    DType = "IPV6"
	DTypeJSON                    DType = "JSON"
	DTypeJSONB                   DType = "JSONB"
	DTypeList                    DType = "LIST"
	DTypeLongBlob                DType = "LONGBLOB"
	DTypeLongText                DType = "LONGTEXT"
	DTypeLowCardinality          DType = "LOWCARDINALITY"
	DTypeMap                     DType = "MAP"
	DTypeMediumBlob              DType = "MEDIUMBLOB"
	DTypeMediumInt               DType = "MEDIUMINT"
	DTypeMediumText              DType = "MEDIUMTEXT"
	DTypeMoney                   DType = "MONEY"
	DTypeName                    DType = "NAME"
	DTypeNChar                   DType = "NCHAR"
	DTypeNested                  DType = "NESTED"
	DTypeNothing                 DType = "NOTHING"
	DTypeNull                    DType = "NULL"
	DTypeNumMultiRange           DType = "NUMMULTIRANGE"
	DTypeNumRange                DType = "NUMRANGE"
	DTypeNVarchar                DType = "NVARCHAR"
	DTypeObject                  DType = "OBJECT"
	DTypeRange                   DType = "RANGE"
	DTypeRowVersion              DType = "ROWVERSION"
	DTypeSerial                  DType = "SERIAL"
	DTypeSet                     DType = "SET"
	DTypeSmallDatetime           DType = "SMALLDATETIME"
	DTypeSmallInt                DType = "SMALLINT"
	DTypeSmallMoney              DType = "SMALLMONEY"
	DTypeSmallSerial             DType = "SMALLSERIAL"
	DTypeStruct                  DType = "STRUCT"
	DTypeSuper                   DType = "SUPER"
	DTypeText                    DType = "TEXT"
	DTypeTinyBlob                DType = "TINYBLOB"
	DTypeTinyText                DType = "TINYTEXT"
	DTypeTime                    DType = "TIME"
	DTypeTimeTz                  DType = "TIMETZ"
	DTypeTimeNs                  DType = "TIME_NS"
	DTypeTimestamp               DType = "TIMESTAMP"
	DTypeTimestampNtz            DType = "TIMESTAMPNTZ"
	DTypeTimestampLtz            DType = "TIMESTAMPLTZ"
	DTypeTimestampTz             DType = "TIMESTAMPTZ"
	DTypeTimestampS              DType = "TIMESTAMP_S"
	DTypeTimestampMs             DType = "TIMESTAMP_MS"
	DTypeTimestampNs             DType = "TIMESTAMP_NS"
	DTypeTinyInt                 DType = "TINYINT"
	DTypeTsMultiRange            DType = "TSMULTIRANGE"
	DTypeTsRange                 DType = "TSRANGE"
	DTypeTstzMultiRange          DType = "TSTZMULTIRANGE"
	DTypeTstzRange               DType = "TSTZRANGE"
	DTypeUBigInt                 DType = "UBIGINT"
	DTypeUInt                    DType = "UINT"
	DTypeUInt128                 DType = "UINT128"
	DTypeUInt256                 DType = "UINT256"
	DTypeUMediumInt              DType = "UMEDIUMINT"
	DTypeUDecimal                DType = "UDECIMAL"
	DTypeUDouble                 DType = "UDOUBLE"
	DTypeUnion                   DType = "UNION"
	DTypeUnknown                 DType = "UNKNOWN"
	DTypeUserDefined             DType = "USER-DEFINED"
	DTypeUSmallInt               DType = "USMALLINT"
	DTypeUTinyInt                DType = "UTINYINT"
	DTypeUUID                    DType = "UUID"
	DTypeVarBinary               DType = "VARBINARY"
	DTypeVarchar                 DType = "VARCHAR"
	DTypeVariant                 DType = "VARIANT"
	DTypeVector                  DType = "VECTOR"
	DTypeXML                     DType = "XML"
	DTypeYear                    DType = "YEAR"
	DTypeTDigest                 DType = "TDIGEST"
)

var dTypeByName = map[string]DType{
	"ARRAY":                   DTypeArray,
	"AGGREGATEFUNCTION":       DTypeAggregateFunction,
	"SIMPLEAGGREGATEFUNCTION": DTypeSimpleAggregateFunction,
	"BIGDECIMAL":              DTypeBigDecimal,
	"BIGINT":                  DTypeBigInt,
	"BIGNUM":                  DTypeBigNum,
	"BIGSERIAL":               DTypeBigSerial,
	"BINARY":                  DTypeBinary,
	"BIT":                     DTypeBit,
	"BLOB":                    DTypeBlob,
	"BOOLEAN":                 DTypeBoolean,
	"BPCHAR":                  DTypeBpchar,
	"CHAR":                    DTypeChar,
	"CHARACTER_SET":           DTypeCharacterSet,
	"DATE":                    DTypeDate,
	"DATE32":                  DTypeDate32,
	"DATEMULTIRANGE":          DTypeDateMultiRange,
	"DATERANGE":               DTypeDateRange,
	"DATETIME":                DTypeDatetime,
	"DATETIME2":               DTypeDatetime2,
	"DATETIME64":              DTypeDatetime64,
	"DECIMAL":                 DTypeDecimal,
	"DECIMAL32":               DTypeDecimal32,
	"DECIMAL64":               DTypeDecimal64,
	"DECIMAL128":              DTypeDecimal128,
	"DECIMAL256":              DTypeDecimal256,
	"DECFLOAT":                DTypeDecFloat,
	"DOUBLE":                  DTypeDouble,
	"DYNAMIC":                 DTypeDynamic,
	"ENUM":                    DTypeEnum,
	"ENUM8":                   DTypeEnum8,
	"ENUM16":                  DTypeEnum16,
	"FILE":                    DTypeFile,
	"FIXEDSTRING":             DTypeFixedString,
	"FLOAT":                   DTypeFloat,
	"GEOGRAPHY":               DTypeGeography,
	"GEOGRAPHYPOINT":          DTypeGeographyPoint,
	"GEOMETRY":                DTypeGeometry,
	"POINT":                   DTypePoint,
	"RING":                    DTypeRing,
	"LINESTRING":              DTypeLineString,
	"MULTILINESTRING":         DTypeMultiLineString,
	"POLYGON":                 DTypePolygon,
	"MULTIPOLYGON":            DTypeMultiPolygon,
	"HLLSKETCH":               DTypeHllSketch,
	"HSTORE":                  DTypeHstore,
	"IMAGE":                   DTypeImage,
	"INET":                    DTypeInet,
	"INT":                     DTypeInt,
	"INT128":                  DTypeInt128,
	"INT256":                  DTypeInt256,
	"INT4MULTIRANGE":          DTypeInt4MultiRange,
	"INT4RANGE":               DTypeInt4Range,
	"INT8MULTIRANGE":          DTypeInt8MultiRange,
	"INT8RANGE":               DTypeInt8Range,
	"INTERVAL":                DTypeInterval,
	"IPADDRESS":               DTypeIpAddress,
	"IPPREFIX":                DTypeIpPrefix,
	"IPV4":                    DTypeIpv4,
	"IPV6":                    DTypeIpv6,
	"JSON":                    DTypeJSON,
	"JSONB":                   DTypeJSONB,
	"LIST":                    DTypeList,
	"LONGBLOB":                DTypeLongBlob,
	"LONGTEXT":                DTypeLongText,
	"LOWCARDINALITY":          DTypeLowCardinality,
	"MAP":                     DTypeMap,
	"MEDIUMBLOB":              DTypeMediumBlob,
	"MEDIUMINT":               DTypeMediumInt,
	"MEDIUMTEXT":              DTypeMediumText,
	"MONEY":                   DTypeMoney,
	"NAME":                    DTypeName,
	"NCHAR":                   DTypeNChar,
	"NESTED":                  DTypeNested,
	"NOTHING":                 DTypeNothing,
	"NULL":                    DTypeNull,
	"NUMMULTIRANGE":           DTypeNumMultiRange,
	"NUMRANGE":                DTypeNumRange,
	"NVARCHAR":                DTypeNVarchar,
	"OBJECT":                  DTypeObject,
	"RANGE":                   DTypeRange,
	"ROWVERSION":              DTypeRowVersion,
	"SERIAL":                  DTypeSerial,
	"SET":                     DTypeSet,
	"SMALLDATETIME":           DTypeSmallDatetime,
	"SMALLINT":                DTypeSmallInt,
	"SMALLMONEY":              DTypeSmallMoney,
	"SMALLSERIAL":             DTypeSmallSerial,
	"STRUCT":                  DTypeStruct,
	"SUPER":                   DTypeSuper,
	"TEXT":                    DTypeText,
	"TINYBLOB":                DTypeTinyBlob,
	"TINYTEXT":                DTypeTinyText,
	"TIME":                    DTypeTime,
	"TIMETZ":                  DTypeTimeTz,
	"TIME_NS":                 DTypeTimeNs,
	"TIMESTAMP":               DTypeTimestamp,
	"TIMESTAMPNTZ":            DTypeTimestampNtz,
	"TIMESTAMPLTZ":            DTypeTimestampLtz,
	"TIMESTAMPTZ":             DTypeTimestampTz,
	"TIMESTAMP_S":             DTypeTimestampS,
	"TIMESTAMP_MS":            DTypeTimestampMs,
	"TIMESTAMP_NS":            DTypeTimestampNs,
	"TINYINT":                 DTypeTinyInt,
	"TSMULTIRANGE":            DTypeTsMultiRange,
	"TSRANGE":                 DTypeTsRange,
	"TSTZMULTIRANGE":          DTypeTstzMultiRange,
	"TSTZRANGE":               DTypeTstzRange,
	"UBIGINT":                 DTypeUBigInt,
	"UINT":                    DTypeUInt,
	"UINT128":                 DTypeUInt128,
	"UINT256":                 DTypeUInt256,
	"UMEDIUMINT":              DTypeUMediumInt,
	"UDECIMAL":                DTypeUDecimal,
	"UDOUBLE":                 DTypeUDouble,
	"UNION":                   DTypeUnion,
	"UNKNOWN":                 DTypeUnknown,
	"USERDEFINED":             DTypeUserDefined,
	"USMALLINT":               DTypeUSmallInt,
	"UTINYINT":                DTypeUTinyInt,
	"UUID":                    DTypeUUID,
	"VARBINARY":               DTypeVarBinary,
	"VARCHAR":                 DTypeVarchar,
	"VARIANT":                 DTypeVariant,
	"VECTOR":                  DTypeVector,
	"XML":                     DTypeXML,
	"YEAR":                    DTypeYear,
	"TDIGEST":                 DTypeTDigest,
}

func DTypeFromName(name string) (DType, bool) {
	d, ok := dTypeByName[name]
	return d, ok
}

func IsType(e Expression, t DType) bool {
	return e != nil && e.Kind() == KindDataType && e.Arg("this") == t
}

func DataType(args Args) Expression      { return newNode(KindDataType, args) }
func DataTypeParam(args Args) Expression { return newNode(KindDataTypeParam, args) }
func Interval(args Args) Expression      { return newNode(KindInterval, args) }
func IntervalSpan(args Args) Expression  { return newNode(KindIntervalSpan, args) }
