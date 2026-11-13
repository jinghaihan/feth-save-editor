using System;
using System.IO;
using System.Linq;
using SaveEditor;
using SaveEditor.Structs;

namespace FethEditor.Core
{
    public sealed class SystemBuffer
    {
        private const int HeaderSize = SystemSave.SIZE_SAVE_HEADER;
        private const int FlagsOffset = HeaderSize + SystemSaveData_V7.COUNT_SAVES * SaveFileInfo.SIZE;
        private readonly byte[] _bytes;
        public int SourceVersion { get; }
        public bool HasInvalidChecksum { get; }

        private SystemBuffer(byte[] bytes, int sourceVersion, bool hasInvalidChecksum)
        {
            _bytes = bytes;
            SourceVersion = sourceVersion;
            HasInvalidChecksum = hasInvalidChecksum;
        }

        public static SystemBuffer Open(string path)
        {
            byte[] bytes = File.ReadAllBytes(path);
            int version = checked((int)BitConverter.ToUInt32(bytes, 4));
            if (bytes.Length != (version == 5 ? SystemSave.SIZE_SAVE_V5 : SystemSave.SIZE_SAVE_V7) ||
                version is not (5 or 7) || BitConverter.ToUInt32(bytes, 8) != bytes.Length)
                throw new InvalidDataException("Expected a version-5 or version-7 system save.");
            bool hasInvalidChecksum = BitConverter.ToUInt32(bytes, 0)
                != Util.CalcChecksum32(bytes.Skip(HeaderSize).ToArray());
            return new SystemBuffer(version == 5 ? UpgradeVersion5(bytes) : bytes,
                version, hasInvalidChecksum);
        }

        private static byte[] UpgradeVersion5(byte[] old)
        {
            var upgraded = new byte[SystemSave.SIZE_SAVE_V7];
            int oldSlotsSize = SystemSaveData_V5.COUNT_SAVES * SaveFileInfo.SIZE;
            int newSlotsSize = SystemSaveData_V7.COUNT_SAVES * SaveFileInfo.SIZE;
            Buffer.BlockCopy(old, HeaderSize, upgraded, HeaderSize, oldSlotsSize);
            for (int slot = SystemSaveData_V5.COUNT_SAVES; slot < SystemSaveData_V7.COUNT_SAVES; slot++)
                BitConverter.GetBytes(0x11u).CopyTo(upgraded, HeaderSize + slot * SaveFileInfo.SIZE);
            Buffer.BlockCopy(old, HeaderSize + oldSlotsSize, upgraded,
                HeaderSize + newSlotsSize, SystemSaveData_V5.SIZE_FLAGS);
            BitConverter.GetBytes(SystemSave.CURRENT_VERSION).CopyTo(upgraded, 4);
            BitConverter.GetBytes(upgraded.Length).CopyTo(upgraded, 8);
            BitConverter.GetBytes(Util.CalcChecksum32(upgraded.Skip(HeaderSize).ToArray()))
                .CopyTo(upgraded, 0);
            return upgraded;
        }

        public SystemSaveData_V7 Data => Util.ReadStructure<SystemSaveData_V7>(
            _bytes.Skip(HeaderSize).ToArray());

        public bool GetFlag(int index)
        {
            CheckFlagIndex(index);
            return (_bytes[FlagsOffset + index / 8] & (1 << (index % 8))) != 0;
        }

        public void SetFlag(int index, bool enabled)
        {
            CheckFlagIndex(index);
            int offset = FlagsOffset + index / 8;
            int mask = 1 << (index % 8);
            _bytes[offset] = enabled ? (byte)(_bytes[offset] | mask) : (byte)(_bytes[offset] & ~mask);
        }

        public byte[] FinishedBytes()
        {
            byte[] result = (byte[])_bytes.Clone();
            BitConverter.GetBytes(Util.CalcChecksum32(result.Skip(HeaderSize).ToArray()))
                .CopyTo(result, 0);
            return result;
        }

        private static void CheckFlagIndex(int index)
        {
            if (index < 0 || index >= SystemSaveData_V7.COUNT_FLAGS)
                throw new ArgumentOutOfRangeException(nameof(index));
        }
    }
}
