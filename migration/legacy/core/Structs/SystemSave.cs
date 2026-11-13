using System;
using System.IO;
using System.Collections.Generic;
using System.Linq;
using System.Runtime.InteropServices;
using System.Text;
using System.Threading.Tasks;

namespace SaveEditor.Structs
{
    [StructLayout(LayoutKind.Sequential, Pack = 1, Size = SIZE)]
    public struct SaveFileInfo
    {
        public const int SIZE = 0x74;

        public uint Flags; //0x7 = used, 0x11 = unused
        public uint Playtime;
        public uint unk2;
        public uint Chapter1;
        public uint Day;
        public uint unk4;
        public uint Chapter2;
        public uint unk5, unk6, unk7, unk8;

        [MarshalAs(UnmanagedType.ByValArray, SizeConst = 0x28)]
        public byte[] PlayerName;

        public uint PlaceId;

        [MarshalAs(UnmanagedType.ByValArray, SizeConst = 0x1C)]
        public byte[] unk9;

        public string GetPlaytime()
        {
            return $"{Playtime / 3600}:{Playtime % 3600 / 60}";
        }

        public string GetPlayerName()
        {
            //1B 4E 30 
            //42 79 6C 65 74 68 
            //1B 4E 31 
            //00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00

            return Util.DecodeString(PlayerName);
        }

        public void Init()
        {
            Flags = 0x11;
            PlayerName = new byte[0x28];
            unk9 = new byte[0x1C];
        }
    }

    [StructLayout(LayoutKind.Sequential, Pack = 1, Size = SIZE)]
    public struct SystemSaveData_V5
    {
        public const int SIZE = 0x440;
        public const int COUNT_SAVES = 7;
        public const int SIZE_FLAGS = 0x114;
        public const int COUNT_FLAGS = SIZE_FLAGS * 8;

        [MarshalAs(UnmanagedType.ByValArray, SizeConst = COUNT_SAVES)]
        public SaveFileInfo[] Infos;

        [MarshalAs(UnmanagedType.ByValArray, SizeConst = SIZE_FLAGS)]
        public byte[] Flags;
    }

    [StructLayout(LayoutKind.Sequential, Pack = 1, Size = SIZE)]
    public struct SystemSaveData_V7
    {
        public const int SIZE = 0x11F8;
        public const int COUNT_SAVES = 37;
        public const int SIZE_FLAGS = 0x134;
        public const int COUNT_FLAGS = SIZE_FLAGS * 8;

        [MarshalAs(UnmanagedType.ByValArray, SizeConst = COUNT_SAVES)]
        public SaveFileInfo[] Infos;

        [MarshalAs(UnmanagedType.ByValArray, SizeConst = SIZE_FLAGS)]
        public byte[] Flags;
    }

    public class SystemSave
    {
        public const int SIZE_SAVE_HEADER = 0xC;
        public const int SIZE_SAVE_V5 = SystemSaveData_V5.SIZE + SIZE_SAVE_HEADER;
        public const int SIZE_SAVE_V7 = SystemSaveData_V7.SIZE + SIZE_SAVE_HEADER;

        public const int CURRENT_VERSION = 7; //1.0.0 = 5, 1.1.0 - 1.20 = 7,
 
        public uint Checksum, CalculatedChecksum;
        public uint SaveVersion;
        public uint SizeOfFile;
        public SystemSaveData_V7 SaveData;
        public bool WarnChecksum, WarnVersionUpdate;

        public SystemSave()
        {
            Checksum = 0;
            SaveVersion = 0xC;
            SizeOfFile = SystemSaveData_V7.SIZE;
            SaveData = new SystemSaveData_V7();
            WarnChecksum = false;
            WarnVersionUpdate = false;
        }

        public void Read(string sPath)
        {
            using (var br = new BinaryReader(File.OpenRead(sPath)))
            {
                Checksum = br.ReadUInt32();
                SaveVersion = br.ReadUInt32();
                SizeOfFile = br.ReadUInt32();

                byte[] data;

                switch (SizeOfFile)
                {
                    case SIZE_SAVE_V5:
                    case SIZE_SAVE_V7:
                        break;

                    default:
                        throw new NotSupportedException($"Current save filesize '{SizeOfFile}' is not supported!");
                }

                switch (SaveVersion)
                {
                    case 5: //v1.0.0 - 1.0.2
                        data = br.ReadBytes(SystemSaveData_V5.SIZE);
                        WarnVersionUpdate = true;
                        break;

                    case 7: //v1.1.0 - 1.2.0
                        data = br.ReadBytes(SystemSaveData_V7.SIZE);
                        break;

                    default:
                        throw new NotSupportedException($"Current save version '{SaveVersion}' is not supported!");
                }

                CalculatedChecksum = Util.CalcChecksum32(data);

                if (CalculatedChecksum != Checksum)
                {
                    WarnChecksum = true;
                }

                try
                {
                    if(SaveVersion < 7)
                    {
                        var old =  Util.ReadStructure<SystemSaveData_V5>(data);
                        ImportOldSave(old);
                    }
                    else
                    {
                        SaveData = Util.ReadStructure<SystemSaveData_V7>(data);
                    }
                }
                catch (Exception e)
                {
                    Console.WriteLine(e);
                    throw;
                }
            }
        }
        
        public void Write(string sPath)
        {
            Util.DeleteFile(sPath);
            
            byte[] data = Util.StructureToByteArray(SaveData);

            using (var bw = new BinaryWriter(File.OpenWrite(sPath)))
            {
                bw.Write(Util.CalcChecksum32(data));
                bw.Write(SaveVersion);
                bw.Write(SizeOfFile);
                bw.Write(data);
            }
        }

        private void ImportOldSave(SystemSaveData_V5 old)
        {
            SaveVersion = CURRENT_VERSION;
            SizeOfFile = SystemSaveData_V7.SIZE;

            SaveData = new SystemSaveData_V7
            {
                Infos = new SaveFileInfo[SystemSaveData_V7.COUNT_SAVES],
                Flags = new byte[SystemSaveData_V7.SIZE_FLAGS]
            };

            old.Infos.CopyTo(SaveData.Infos, 0);
            old.Flags.CopyTo(SaveData.Flags, 0);

            //init new infos
            for(int i = 0; i < SaveData.Infos.Length; i++)
            {
                var info = SaveData.Infos[i];

                if(info.Flags == 0)
                {
                    info.Init();
                }

                SaveData.Infos[i] = info;
            }
        }

    }
}
