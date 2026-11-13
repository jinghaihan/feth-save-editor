using System;
using System.Collections.Generic;
using System.Linq;
using System.Runtime.InteropServices;
using System.Text;
using System.Threading.Tasks;

namespace SaveEditor.Structs
{
    public struct Player_V11
    {
        //Debug Only version? some leftover code can be found in Player::ImportOldSave //@.text:00000000003E94F0 in Update 1.2.0
    }

    [StructLayout(LayoutKind.Sequential, Pack = 1, Size = SIZE)]
    public struct Player_V13
    {
        public const int SIZE = 0x1EC8;
        public const int COUNT_SUPPORT = 256;

        [MarshalAs(UnmanagedType.ByValArray, SizeConst = 0x18)]
        public byte[] field_0;

        [MarshalAs(UnmanagedType.ByValArray, SizeConst = 5)]
        public CharacterData_V23[] OnlineCharacter; //0x18

        [MarshalAs(UnmanagedType.ByValArray, SizeConst = 48)]
        public byte[] field_888; //0x888
        
        [MarshalAs(UnmanagedType.ByValArray, SizeConst = 48)]
        public byte[] field_8B8; //0x8B8

        [MarshalAs(UnmanagedType.ByValArray, SizeConst = 48)]
        public byte[] field_8E8; //0x8E8

        [MarshalAs(UnmanagedType.ByValArray, SizeConst = 48)]
        public byte[] field_918; //0x918
        
        [MarshalAs(UnmanagedType.ByValArray, SizeConst = 232)]
        public byte[] field_948; //0x948

        [MarshalAs(UnmanagedType.ByValArray, SizeConst = 200)]
        public Battalion[] Battalions; //0xA30

        public uint Playtime, Money, field_1078, Chapter; //0x1070

        [MarshalAs(UnmanagedType.ByValArray, SizeConst = COUNT_SUPPORT)]
        public ushort[] CharacterSupportValues; //0x1080

        public byte Difficulty, Gamestyle, Route, field_1283, MapID, Flag0;

        [MarshalAs(UnmanagedType.ByValArray, SizeConst = 271)]
        public byte[] Flags;

        [MarshalAs(UnmanagedType.ByValArray, SizeConst = 223)]
        public byte[] MiscItems; //0x1395

        [MarshalAs(UnmanagedType.ByValArray, SizeConst = 225)]
        public byte[] GiftItems; //0x1474

        [MarshalAs(UnmanagedType.ByValArray, SizeConst = 5)]
        public byte[] field_1555;
        
        [MarshalAs(UnmanagedType.ByValArray, SizeConst = 256)]
        public short[] field_155A;

        [MarshalAs(UnmanagedType.ByValArray, SizeConst = 20)]
        public short[] field_175A;
        
        [MarshalAs(UnmanagedType.ByValArray, SizeConst = 1860)]
        public byte[] field_1782;

        public short field_1EC6;

        public string GetPlaytime()
        {
            return $"{Playtime / 3600}:{Playtime % 3600 / 60}";
        }
    }

    [StructLayout(LayoutKind.Sequential, Pack = 1, Size = SIZE)]
    public struct Player_V23
    {
        public const int SIZE = 0x1EC8; //1.0.0: 0x1EC8, 1.1.0: 0x1EC8

        public const int COUNT_BATTALION = 200;
        public const int COUNT_SUPPORT = 270; //1.0.0: 256, 1.1.0: 270
        public const int COUNT_FLAGS = 271;
        public const int COUNT_MISC_ITEMS = 223;
        public const int COUNT_GIFT_ITEMS1 = 225;
        public const int COUNT_GIFT_ITEMS2 = 20;
        public const int COUNT_GIFT_ITEMS = COUNT_GIFT_ITEMS1 + COUNT_GIFT_ITEMS2;

        [MarshalAs(UnmanagedType.ByValArray, SizeConst = 0x18)]
        public byte[] field_0;

        [MarshalAs(UnmanagedType.ByValArray, SizeConst = 5)]
        public CharacterDataPart1[] OnlineCharacter; //0x18

        [MarshalAs(UnmanagedType.ByValArray, SizeConst = 48)]
        public byte[] field_888; //0x888
        
        [MarshalAs(UnmanagedType.ByValArray, SizeConst = 48)]
        public byte[] field_8B8; //0x8B8

        [MarshalAs(UnmanagedType.ByValArray, SizeConst = 48)]
        public byte[] field_8E8; //0x8E8

        [MarshalAs(UnmanagedType.ByValArray, SizeConst = 48)]
        public byte[] field_918; //0x918
        
        [MarshalAs(UnmanagedType.ByValArray, SizeConst = 232)]
        public byte[] field_948; //0x948

        [MarshalAs(UnmanagedType.ByValArray, SizeConst = COUNT_BATTALION)]
        public Battalion[] Battalions; //0xA30

        public uint Playtime, Money, field_1078, Chapter; //0x1070

        [MarshalAs(UnmanagedType.ByValArray, SizeConst = COUNT_SUPPORT)]
        public ushort[] CharacterSupportValues; //0x1080

        public byte Difficulty, Gamestyle, Route;
        public byte field_129F;
        public byte MapID;
        public byte field_12A1;

        [MarshalAs(UnmanagedType.ByValArray, SizeConst = COUNT_FLAGS)]
        public byte[] Flags;

        [MarshalAs(UnmanagedType.ByValArray, SizeConst = COUNT_MISC_ITEMS)]
        public byte[] MiscItems;

        [MarshalAs(UnmanagedType.ByValArray, SizeConst = COUNT_GIFT_ITEMS1)]
        public byte[] GiftItems;

        public uint field_1571;
        public byte field_1575;
        
        [MarshalAs(UnmanagedType.ByValArray, SizeConst = 270)] //1.0.0: 256, 1.1.0: 270
        public short[] field_1576;

        [MarshalAs(UnmanagedType.ByValArray, SizeConst = 30)]
        public short[] field_1792;

        public byte field_17CE;
        public byte field_17CF;
        public byte field_17D0;
        public ushort field_17D1;
        public byte field_17D3;
        public uint field_17D4;

        [MarshalAs(UnmanagedType.ByValArray, SizeConst = 385)]
        public byte[] field_17D8;
        
        [MarshalAs(UnmanagedType.ByValArray, SizeConst = 110)] //10 * 11 DLC character skill ranks
        public byte[] field_1959;

        [MarshalAs(UnmanagedType.ByValArray, SizeConst = 455)] //35 * 13 class mastery flags
        public byte[] field_19C7;

        [MarshalAs(UnmanagedType.ByValArray, SizeConst = 130)] //10 * 13 additional character class mastery flags
        public byte[] field_1B8E;

        public uint field_1C10;

        [MarshalAs(UnmanagedType.ByValArray, SizeConst = 383)]
        public byte[] field_1C14;
        
        [MarshalAs(UnmanagedType.ByValArray, SizeConst = COUNT_GIFT_ITEMS2)] //new in 1.1.0, why did they split it instead of increasing the size?
        public byte[] GiftItems2;
        
        public byte field_1DA7;
        public ushort field_1DA8;
        public byte field_1DAA;
        
        [MarshalAs(UnmanagedType.ByValArray, SizeConst = 200)]
        public byte[] field_1DAB;

        public byte field_1E73;
               
        [MarshalAs(UnmanagedType.ByValArray, SizeConst = 84)]
        public byte[] padding_1E74;

        public string GetPlaytime()
        {
            return $"{Playtime / 3600}:{Playtime % 3600 / 60}";
        }

        public byte GetGiftItem(int idx)
        {
            if(idx < COUNT_GIFT_ITEMS1)
                return GiftItems[idx];
            else if(idx - COUNT_GIFT_ITEMS1 < COUNT_GIFT_ITEMS2)
                return GiftItems2[idx - COUNT_GIFT_ITEMS1];

            return 0;
        }
        
        public void SetGiftItem(int idx, byte amount)
        {
            if(idx < COUNT_GIFT_ITEMS1)
                GiftItems[idx] = amount;
            else if(idx - COUNT_GIFT_ITEMS1 < COUNT_GIFT_ITEMS2)
                GiftItems2[idx - COUNT_GIFT_ITEMS1] = amount;
        }


    }


}
