using System;
using System.Collections.Generic;
using System.Text.Json;
using System.Text.Json.Serialization;

namespace FethEditor.Cli
{
    internal sealed class JsonCompat
    {
        public int MaxJsonLength { get; set; }
        public int RecursionLimit { get; set; } = 100;

        public string Serialize(object value)
        {
            var options = new JsonSerializerOptions { MaxDepth = RecursionLimit };
            options.Converters.Add(new ByteArrayAsNumbers());
            return JsonSerializer.Serialize(value, options);
        }

        public object DeserializeObject(string json)
        {
            using var document = JsonDocument.Parse(json, new JsonDocumentOptions { MaxDepth = RecursionLimit });
            return ConvertElement(document.RootElement);
        }

        private static object ConvertElement(JsonElement element)
        {
            switch (element.ValueKind)
            {
                case JsonValueKind.Object:
                    var fields = new Dictionary<string, object>();
                    foreach (var property in element.EnumerateObject())
                        fields.Add(property.Name, ConvertElement(property.Value));
                    return fields;
                case JsonValueKind.Array:
                    var values = new List<object>();
                    foreach (var value in element.EnumerateArray())
                        values.Add(ConvertElement(value));
                    return values.ToArray();
                case JsonValueKind.String:
                    return element.GetString();
                case JsonValueKind.Number:
                    return element.TryGetInt64(out long integer) ? integer : element.GetDouble();
                case JsonValueKind.True:
                    return true;
                case JsonValueKind.False:
                    return false;
                case JsonValueKind.Null:
                    return null;
                default:
                    throw new FormatException("Unsupported JSON value.");
            }
        }

        private sealed class ByteArrayAsNumbers : JsonConverter<byte[]>
        {
            public override byte[] Read(ref Utf8JsonReader reader, Type typeToConvert,
                JsonSerializerOptions options) => throw new NotSupportedException();

            public override void Write(Utf8JsonWriter writer, byte[] value, JsonSerializerOptions options)
            {
                writer.WriteStartArray();
                foreach (byte number in value) writer.WriteNumberValue(number);
                writer.WriteEndArray();
            }
        }
    }
}
