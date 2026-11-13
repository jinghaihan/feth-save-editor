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
    public struct SaveData_V13
    {
        public const int SIZE = 0x25400;
        public const int ITEM_COUNT = 400;
        public const int CHARACTER_COUNT = 60;
        public const int NPC_COUNT = 500;

        [MarshalAs(UnmanagedType.ByValArray, SizeConst = ITEM_COUNT)]
        public Item[] Items; //0x0

        public uint ItemCount; //0x640

        [MarshalAs(UnmanagedType.ByValArray, SizeConst = CHARACTER_COUNT)]
        public Character_V13[] Characters; //0x644

        [MarshalAs(UnmanagedType.ByValArray, SizeConst = 0x28)]
        public byte[] PlayerName; //0x8984

        [MarshalAs(UnmanagedType.ByValArray, SizeConst = 0xC)]
        public byte[] field_89AC; //0x89AC

        public uint SizeOfNpcs; //0x89B8 //19DF0

        [MarshalAs(UnmanagedType.ByValArray, SizeConst = NPC_COUNT)]
        public NPC[] NPCs; //0x89BC
        [MarshalAs(UnmanagedType.ByValArray, SizeConst = 0x7B0)]
        public byte[] field_21FFC; //0x21FFC

        [MarshalAs(UnmanagedType.ByValArray, SizeConst = 0x30D)]
        public byte[] field_227AC; //0x227AC

        public Player_V13 Player; //0x22AB9
        
        public Activities_V13 Activities; //0x2498D
    }

    [StructLayout(LayoutKind.Sequential, Pack = 1, Size = SIZE)]
    public struct SaveData_V23
    {
        public const int SIZE = 0x25B20;
        public const int ITEM_COUNT = 400;
        public const int CHARACTER_COUNT = 60;
        public const int NPC_COUNT = 500;

        [MarshalAs(UnmanagedType.ByValArray, SizeConst = ITEM_COUNT)]
        public Item[] Items; //0x0

        public uint ItemCount; //0x640

        [MarshalAs(UnmanagedType.ByValArray, SizeConst = CHARACTER_COUNT)]
        public Character_V23[] Characters; //0x644

        [MarshalAs(UnmanagedType.ByValArray, SizeConst = 0x28)]
        public byte[] PlayerName; //1.0.0: 0x8984, 1.1.0: 0x9014

        [MarshalAs(UnmanagedType.ByValArray, SizeConst = 0xC)]
        public byte[] field_89AC; //1.0.0: 0x89AC, 1.1.0: 0x903C

        public uint SizeOfNpcs; //1.0.0: 0x89B8, 1.1.0: 0x9048 //19DF0

        [MarshalAs(UnmanagedType.ByValArray, SizeConst = NPC_COUNT)]
        public NPC[] NPCs; //1.0.0: 0x89BC, 1.1.0: 0x904C
        [MarshalAs(UnmanagedType.ByValArray, SizeConst = 0x7B0)]
        public byte[] field_21FFC; //1.0.0: 0x21FFC, 1.1.0: 0x2268C

        [MarshalAs(UnmanagedType.ByValArray, SizeConst = 0x39D)] //changed from 0x30D to 0x39D
        public byte[] field_227AC; //1.0.0: 0x227AC, 1.1.0: 0x22E3C

        public Player_V23 Player; //1.0.0: 0x22AB9, 1.1.0: 0x231D9
        
        public Activities_V23 Activities; //1.0.0: 0x24981, 1.1.0: 0x250A1
    }

    public class Save
    {
        public const int SIZE_SAVE_HEADER = 0xC;
        public const int SIZE_SAVE_V13 = SaveData_V13.SIZE + SIZE_SAVE_HEADER;
        public const int SIZE_SAVE_V23 = SaveData_V23.SIZE + SIZE_SAVE_HEADER;
 
        public const int CURRENT_VERSION = 23; // 1.1.0 - 1.20 = 23,

        public uint Checksum, CalculatedChecksum;
        public uint SaveVersion;
        public uint SizeOfFile;
        public SaveData_V23 SaveData;
        public bool WarnChecksum, WarnVersionUpdate;

        public Save()
        {
            Checksum = 0;
            SaveVersion = 0xC;
            SizeOfFile = SaveData_V23.SIZE;
            SaveData = new SaveData_V23();
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
                    case SIZE_SAVE_V13:
                    case SIZE_SAVE_V23:
                        break;

                    default:
                        throw new NotSupportedException($"Current save filesize '{SizeOfFile}' is not supported!");
                }

                switch (SaveVersion)
                {
                    //version 11 and lower are debug only versions and never released to the public

                    case 12: //v1.0.0
                    case 13: //v1.0.1 - 1.0.2
                        data = br.ReadBytes(SaveData_V13.SIZE);
                        throw new NotSupportedException($"Current save version '{SaveVersion}' is not supported!");

                    case 23: //v1.1.0 - 1.2.0
                        data = br.ReadBytes(SaveData_V23.SIZE);
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
                    if(SaveVersion < 23)
                    {
                        var old =  Util.ReadStructure<SaveData_V13>(data);
                        ImportOldSave(old);
                    }
                    else
                    {
                        SaveData = Util.ReadStructure<SaveData_V23>(data);
                    }
                    
                    FixImpossibleValues();
                }
                catch (Exception e)
                {
                    Console.WriteLine(e);
                    throw;
                }
            }
        }

        public void FixImpossibleValues()
        {
            //fix character class exp
            for (int i = 0; i < Database.CHARACTER_COUNT; i++)
            {
                var character = SaveData.Characters[i];

                for (int j = 0; j < Database.MAX_CLASS; j++)
                {
                    character.data.ClassExp[j] = (ushort)Math.Min(character.data.ClassExp[j], Database.GetMaxClassExp(j));
                }

                character.data.CurrentClassExp = (ushort)Math.Min(character.data.CurrentClassExp, Database.GetMaxClassExp(character.Class));
                SaveData.Characters[i] = character;
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

        private void ImportOldSave(SaveData_V13 old)
        {
            throw new NotImplementedException();

            SaveVersion = CURRENT_VERSION;
            SizeOfFile = SaveData_V23.SIZE;

            WarnVersionUpdate = true;

            //create empty save
            SaveData = new SaveData_V23
            {
                Items = new Item[SaveData_V23.ITEM_COUNT],
                ItemCount = 0,
                Characters = new Character_V23[SaveData_V23.CHARACTER_COUNT],
                PlayerName = new byte[0x28],
                field_89AC = new byte[0xC],
                SizeOfNpcs = 0,
                NPCs = new NPC[SaveData_V23.NPC_COUNT],
                field_21FFC = new byte[0x7B0],
                field_227AC = new byte[0x30D],

                Player = new Player_V23(),
                Activities = new Activities_V23()
            };

            //fill data from old save
            old.Items.CopyTo(SaveData.Items, 0);
            SaveData.ItemCount = old.ItemCount;
            old.Characters.CopyTo(SaveData.Characters, 0);
            old.PlayerName.CopyTo(SaveData.PlayerName, 0);
            old.field_89AC.CopyTo(SaveData.field_89AC, 0);
            SaveData.SizeOfNpcs = old.SizeOfNpcs;
            old.NPCs.CopyTo(SaveData.NPCs, 0);
            old.field_21FFC.CopyTo(SaveData.field_21FFC, 0);
            old.field_227AC.CopyTo(SaveData.field_227AC, 0);

            var oldPlayer = Util.StructureToByteArray(old.Player);
            var oldActivities = Util.StructureToByteArray(old.Activities); 

            byte[] newPlayer = new byte[Player_V23.SIZE];
            byte[] newActivities = new byte[Activities_V23.SIZE];

            Array.Copy(oldPlayer, 0, newPlayer, 0, 0xA30);
            CopyOldToNew(old.Player, SaveData.Player, oldPlayer, newPlayer, nameof(old.Player.Battalions), Marshal.SizeOf(old.Player.Battalions));
            CopyOldToNew(old.Player, SaveData.Player, oldPlayer, newPlayer, nameof(old.Player.CharacterSupportValues), Marshal.SizeOf(old.Player.CharacterSupportValues));
            CopyOldToNew(old.Player, SaveData.Player, oldPlayer, newPlayer, nameof(old.Player.Flags), Marshal.SizeOf(old.Player.Flags));
            CopyOldToNew(old.Player, SaveData.Player, oldPlayer, newPlayer, nameof(old.Player.MiscItems), Marshal.SizeOf(old.Player.MiscItems));
            CopyOldToNew(old.Player, SaveData.Player, oldPlayer, newPlayer, nameof(old.Player.GiftItems), Marshal.SizeOf(old.Player.GiftItems));

            //TODO: finish player import

            Array.Copy(oldActivities, 0, newActivities, 0, 0x3F2);
            Array.Copy(oldActivities, 0x3F2, newActivities, 0x3F4, 0x68B);

            SaveData.Player = Util.ReadStructure<Player_V23>(newPlayer);
            SaveData.Activities = Util.ReadStructure<Activities_V23>(newActivities);



        }

        private void CopyOldToNew(Player_V13 oldP, Player_V23 newP, byte[] oldStruct, byte[] newStruct, string Name, int Size)
        {
            int oldOffset = (int)Marshal.OffsetOf<Player_V13>(Name);
            int newOffset = (int)Marshal.OffsetOf<Player_V23>(Name);

            Array.Copy(oldStruct, oldOffset, newStruct, newOffset, Size);
        }









    }
}
