using System;
using System.IO;
using System.Linq;

namespace SaveEditor.Properties
{
    internal static class Resources
    {
        internal static byte[] fixed_persondata_bin => Read("fixed_persondata.bin.gz");
        internal static byte[] fixed_classdata_bin => Read("fixed_classdata.bin.gz");
        internal static byte[] fixed_data_bin => Read("fixed_data.bin.gz");
        internal static byte[] msgdata_bin => Read("msgdata.bin.gz");

        private static byte[] Read(string name)
        {
            var assembly = typeof(Resources).Assembly;
            string[] available = assembly.GetManifestResourceNames();
            string resourceName = available.SingleOrDefault(candidate =>
                candidate.Equals(name, StringComparison.OrdinalIgnoreCase) ||
                candidate.EndsWith("." + name, StringComparison.OrdinalIgnoreCase));
            string sidecar = Path.Combine(AppContext.BaseDirectory, "Database", name);
            using Stream stream = resourceName is not null
                ? assembly.GetManifestResourceStream(resourceName)
                    ?? throw new InvalidDataException("Could not open embedded database: " + resourceName)
                : File.Exists(sidecar)
                    ? File.OpenRead(sidecar)
                    : throw new InvalidDataException("Missing game database: " + name
                        + "; checked manifest and " + sidecar);
            using var memory = new MemoryStream();
            stream.CopyTo(memory);
            return memory.ToArray();
        }
    }
}
