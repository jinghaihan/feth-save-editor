using System;
using System.IO;

namespace FethEditor.Core
{
    public static class VerifiedFileWriter
    {
        public static string Write(string destination, byte[] output, Func<string, bool> verify)
        {
            destination = Path.GetFullPath(destination);
            string temporary = destination + ".tmp-" + Guid.NewGuid().ToString("N");
            string backup = null;
            try
            {
                File.WriteAllBytes(temporary, output);
                if (!verify(temporary))
                    throw new InvalidDataException("Written save failed verification.");

                if (File.Exists(destination))
                {
                    backup = destination + ".backup-" + DateTime.UtcNow.ToString("yyyyMMddTHHmmssfffZ")
                        + "-" + Guid.NewGuid().ToString("N");
                    File.Replace(temporary, destination, backup);
                }
                else
                    File.Move(temporary, destination);
                return backup;
            }
            finally
            {
                if (File.Exists(temporary)) File.Delete(temporary);
            }
        }
    }
}
