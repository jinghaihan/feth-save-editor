using System;
using System.Collections.Generic;
using System.Linq;
using System.Runtime.InteropServices;
using System.Text;
using System.Threading.Tasks;

namespace SaveEditor.Structs
{
    [StructLayout(LayoutKind.Sequential, Pack = 1, Size = SIZE)]
    public struct Activities_V13
    {
        public const int SIZE = 0xA7F;

        public int field_0, field_4, field_8;

        public uint Reputation; //0xC
        public ushort field_10, InstructExp;
        
        [MarshalAs(UnmanagedType.ByValArray, SizeConst = 9)]
        public short[] field_14;

        public byte field_26, ActivityExplore, ActivityLesson, ActivityBattle;
        public byte field_2A, Statue1, Statue2, Statue3, Statue4, field_2F;

        [MarshalAs(UnmanagedType.ByValArray, SizeConst = 200)]
        public byte[] field_30;
        
        [MarshalAs(UnmanagedType.ByValArray, SizeConst = 100)]
        public byte[] field_F8;
           
        [MarshalAs(UnmanagedType.ByValArray, SizeConst = 50)]
        public byte[] field_15C;

        public byte field_18E;

        [MarshalAs(UnmanagedType.ByValArray, SizeConst = 200)]
        public byte[] field_18F;

        public byte field_257;
        
        [MarshalAs(UnmanagedType.ByValArray, SizeConst = 50)]
        public byte[] field_258;
        
        [MarshalAs(UnmanagedType.ByValArray, SizeConst = 200)]
        public byte[] field_28A;
           
        [MarshalAs(UnmanagedType.ByValArray, SizeConst = 10)]
        public byte[] field_352;
               
        [MarshalAs(UnmanagedType.ByValArray, SizeConst = 10)]
        public byte[] field_35C;
           
        [MarshalAs(UnmanagedType.ByValArray, SizeConst = 100)]
        public byte[] field_366;
              
        [MarshalAs(UnmanagedType.ByValArray, SizeConst = 40)]
        public byte[] field_3CA;

        [MarshalAs(UnmanagedType.ByValArray, SizeConst = 2)]
        public byte[] field_3F2; //new

        [MarshalAs(UnmanagedType.ByValArray, SizeConst = 150)]
        public byte[] QuestStateList;

        public byte field_488;

        [MarshalAs(UnmanagedType.ByValArray, SizeConst = 3)]
        public byte[] field_489;

        [MarshalAs(UnmanagedType.ByValArray, SizeConst = 3)]
        public byte[] field_48C;

        [MarshalAs(UnmanagedType.ByValArray, SizeConst = 66)]
        public byte[] field_48F;
        
        [MarshalAs(UnmanagedType.ByValArray, SizeConst = 76)]
        public byte[] field_4D1;
             
        [MarshalAs(UnmanagedType.ByValArray, SizeConst = 41)]
        public byte[] field_51D;

        public uint field_546;

        [MarshalAs(UnmanagedType.ByValArray, SizeConst = 17)]
        public byte[] field_54A;

        public byte PlayLog_Wark, PlayLog_Lecture, PlayLog_ToBtl, PlayLog_Rest, PlayLog_Trnmnt, PlayLog_Sing;
        public byte PlayLog_Lunch, PlayLog_Cooking, PlayLog_Drill, PlayLog_Teaparty, PlayLog_SCOUT;

        public int GetInstructLevel()
        {
            return Database.GetProfessorRankFromExperience(InstructExp);
        }

        public string GetInstructRank()
        {
            int level = GetInstructLevel();
            return ((enmRank)level).GetDescription();
        }


    }

    [StructLayout(LayoutKind.Sequential, Pack = 1, Size = SIZE)]
    public struct Activities_V23
    {
        public const int SIZE = 0xA7F;

        public int field_0, field_4, field_8;

        public uint Reputation; //0xC
        public ushort field_10, InstructExp;
        
        [MarshalAs(UnmanagedType.ByValArray, SizeConst = 9)]
        public short[] field_14;

        public byte field_26, ActivityExplore, ActivityLesson, ActivityBattle;
        public byte field_2A, Statue1, Statue2, Statue3, Statue4, field_2F;

        [MarshalAs(UnmanagedType.ByValArray, SizeConst = 200)]
        public byte[] field_30;
        
        [MarshalAs(UnmanagedType.ByValArray, SizeConst = 100)]
        public byte[] field_F8;
           
        [MarshalAs(UnmanagedType.ByValArray, SizeConst = 50)]
        public byte[] field_15C;

        public byte field_18E;

        [MarshalAs(UnmanagedType.ByValArray, SizeConst = 200)]
        public byte[] field_18F;

        public byte field_257;
        
        [MarshalAs(UnmanagedType.ByValArray, SizeConst = 50)]
        public byte[] field_258;
        
        [MarshalAs(UnmanagedType.ByValArray, SizeConst = 200)]
        public byte[] field_28A;
           
        [MarshalAs(UnmanagedType.ByValArray, SizeConst = 10)]
        public byte[] field_352;
               
        [MarshalAs(UnmanagedType.ByValArray, SizeConst = 10)]
        public byte[] field_35C;
           
        [MarshalAs(UnmanagedType.ByValArray, SizeConst = 100)]
        public byte[] field_366;
              
        [MarshalAs(UnmanagedType.ByValArray, SizeConst = 40)]
        public byte[] field_3CA;

        [MarshalAs(UnmanagedType.ByValArray, SizeConst = 2)]
        public byte[] field_3F2; //new

        [MarshalAs(UnmanagedType.ByValArray, SizeConst = 150)]
        public byte[] QuestStateList;

        public byte field_48A;

        [MarshalAs(UnmanagedType.ByValArray, SizeConst = 3)]
        public byte[] field_48B;

        [MarshalAs(UnmanagedType.ByValArray, SizeConst = 3)]
        public byte[] field_48E;

        [MarshalAs(UnmanagedType.ByValArray, SizeConst = 66)]
        public byte[] field_491;
        
        [MarshalAs(UnmanagedType.ByValArray, SizeConst = 76)]
        public byte[] field_4D3;
             
        [MarshalAs(UnmanagedType.ByValArray, SizeConst = 41)]
        public byte[] field_51F;

        public uint field_548;

        [MarshalAs(UnmanagedType.ByValArray, SizeConst = 17)]
        public byte[] field_54A;

        public byte PlayLog_Wark, PlayLog_Lecture, PlayLog_ToBtl, PlayLog_Rest, PlayLog_Trnmnt, PlayLog_Sing;
        public byte PlayLog_Lunch, PlayLog_Cooking, PlayLog_Drill, PlayLog_Teaparty, PlayLog_SCOUT;
        
        [MarshalAs(UnmanagedType.ByValArray, SizeConst = 1303)]
        public byte[] field_568;

        public int GetInstructLevel()
        {
            return Database.GetProfessorRankFromExperience(InstructExp);
        }

        public string GetInstructRank()
        {
            int level = GetInstructLevel();
            return ((enmRank)level).GetDescription();
        }


    }

}
