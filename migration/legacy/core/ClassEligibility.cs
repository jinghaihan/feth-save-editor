using System;
using SaveEditor;

namespace FethEditor.Core
{
    public static class ClassEligibility
    {
        public static int RecordForUnit(int unitId)
        {
            var entries = Database.BinaryDatabase?.CharacterEntries;
            if (entries == null || unitId < 0 || unitId >= entries.Count)
                return -1;

            int record = entries[unitId].MainCharacterId;
            if (record < 0 && unitId <= 34)
                record = unitId;
            return IsPlayableRecord(record) ? record : -1;
        }

        public static bool IsAvailable(int record, int classId)
        {
            if (!IsPlayableRecord(record) || classId < 0 || classId >= Database.MAX_CLASS)
                return false;

            // The playable intermediate Pegasus Knight is 54; 23 is a separate advanced variant.
            if (classId == 23)
                return false;
            if (classId is 0 or 1)
                return classId == StartingClass(record);
            if (classId == 6)
                return record is 2 or 3 or 4;
            if (classId == 17 || classId == 58)
                return record == 4;
            if (classId == 40 || classId == 56)
                return record == 2;
            if (classId == 42)
                return record is 0 or 1;
            if (classId == 43)
                return CanEnterWhiteHeronCup(record);
            if (classId == 44 || classId == 57)
                return record == 3;
            if (classId == 91)
                return record == 43;

            bool female = IsFemale(record);
            if (classId is 13 or 15 or 18 or 27 or 29 or 38)
                return !female;
            if (classId is 31 or 39 or 54 or 86 or 87)
                return female;
            return classId is >= 2 and <= 39 or 84 or 85;
        }

        public static bool IsPlayableRecord(int record) =>
            record is >= 0 and <= 34 or >= 38 and <= 41 or 43 or 44;

        private static bool CanEnterWhiteHeronCup(int record) =>
            record is >= 2 and <= 25 or 27 or >= 38 and <= 41;

        private static int StartingClass(int record)
        {
            if (record is 0 or 1)
                return 1;
            // Faculty and Jeritza join in promoted classes, but retain Noble/Commoner as their base class.
            if (record is 26 or 27 or 28 or 30 or 31 or 32 or 43)
                return 0;
            if (record is 29 or 33)
                return 1;
            int unit = UnitForRecord(record);
            return unit < 0 ? -1 : Database.BinaryDatabase.CharacterEntries[unit].BaseClass;
        }

        private static bool IsFemale(int record)
        {
            if (record is 0 or 1)
                return record == 1;
            int unit = UnitForRecord(record);
            if (unit < 0)
                throw new InvalidOperationException($"No database character maps to record {record}.");
            return Database.BinaryDatabase.CharacterEntries[unit].Gender != 0;
        }

        private static int UnitForRecord(int record)
        {
            var entries = Database.BinaryDatabase?.CharacterEntries
                ?? throw new InvalidOperationException("The character database has not been loaded.");
            for (int unit = 0; unit < entries.Count; unit++)
                if (entries[unit].MainCharacterId == record)
                    return unit;
            return -1;
        }
    }
}
