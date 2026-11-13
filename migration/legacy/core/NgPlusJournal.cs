using System;
using System.Collections.Generic;
using System.IO;
using System.Linq;
using SaveEditor;
using SaveEditor.Structs;

namespace FethEditor.Core
{
    public sealed class NgPlusJournal
    {
        public const int CharacterCount = 45;
        public const int SkillCount = 11;
        public const int MaxSkillRank = 11;
        public const int ClassCount = 100;
        private const int ProfessorOffset = 0x17CE;
        private const int SupportOffset = 0x1576;
        private const int SkillOffset = 0x17D8;
        // The 45 x 11 skill ranks end at 0x19C7. Class flags begin immediately
        // afterward, including five bytes currently grouped with field_1959.
        private const int ClassOffset = SkillOffset + CharacterCount * SkillCount;
        private const int ClassBytesPerCharacter = 13;

        private readonly byte[] file;
        private readonly int player;

        public NgPlusJournal(byte[] file, int playerOffset)
        {
            this.file = file ?? throw new ArgumentNullException(nameof(file));
            player = playerOffset;
            if (player < 0 || player + ClassOffset + CharacterCount * ClassBytesPerCharacter > file.Length)
                throw new InvalidDataException("The NG+ journal extends beyond the save file.");
        }

        public int ProfessorRank => file[player + ProfessorOffset];

        public string GetCharacterName(int recordIndex)
        {
            CheckIndex(recordIndex, CharacterCount, nameof(recordIndex));
            return CharacterName(recordIndex);
        }

        public string GetSupportName(int index)
        {
            CheckIndex(index, Player_V23.COUNT_SUPPORT, nameof(index));
            return SupportName(index);
        }

        public int GetSupportPoints(int index)
        {
            CheckIndex(index, Player_V23.COUNT_SUPPORT, nameof(index));
            int offset = player + SupportOffset + index * 2;
            return file[offset] | file[offset + 1] << 8;
        }

        public int GetSkillRank(int recordIndex, int skillIndex)
        {
            CheckIndex(recordIndex, CharacterCount, nameof(recordIndex));
            CheckIndex(skillIndex, SkillCount, nameof(skillIndex));
            return file[player + SkillOffset + recordIndex * SkillCount + skillIndex];
        }

        public bool IsClassMastered(int recordIndex, int classId)
        {
            CheckIndex(recordIndex, CharacterCount, nameof(recordIndex));
            CheckIndex(classId, ClassCount, nameof(classId));
            var (offset, mask) = ClassBit(recordIndex, classId);
            return (file[offset] & mask) != 0;
        }

        public void SetProfessorRank(int rank)
        {
            if (rank < 0 || rank > 9) throw new ArgumentOutOfRangeException(nameof(rank));
            file[player + ProfessorOffset] = (byte)rank;
        }

        public void SetSupportPoints(int index, int points)
        {
            CheckIndex(index, Player_V23.COUNT_SUPPORT, nameof(index));
            if (points < 0 || points > ushort.MaxValue) throw new ArgumentOutOfRangeException(nameof(points));
            int offset = player + SupportOffset + index * 2;
            file[offset] = (byte)points;
            file[offset + 1] = (byte)(points >> 8);
        }

        public void ReachMaxSupportRank(int index)
        {
            int target = SupportPairRanks.MaxRankPoints(index);
            SetSupportPoints(index, Math.Max(GetSupportPoints(index), target));
        }

        public int ReachMaxSupportRanks()
        {
            int updated = 0;
            for (int index = 0; index < Player_V23.COUNT_SUPPORT; index++)
            {
                if (SupportPairRanks.MaxRank(index) == "None"
                    || GetSupportPoints(index) >= SupportPairRanks.MaxRankPoints(index)) continue;
                ReachMaxSupportRank(index);
                updated++;
            }
            return updated;
        }

        public void SetSkillRank(int recordIndex, int skillIndex, int rank)
        {
            CheckIndex(recordIndex, CharacterCount, nameof(recordIndex));
            CheckIndex(skillIndex, SkillCount, nameof(skillIndex));
            if (rank < 0 || rank > MaxSkillRank) throw new ArgumentOutOfRangeException(nameof(rank));
            file[player + SkillOffset + recordIndex * SkillCount + skillIndex] = (byte)rank;
        }

        public void SetClassMastered(int recordIndex, int classId, bool mastered)
        {
            var (offset, mask) = ClassBit(recordIndex, classId);
            file[offset] = mastered ? (byte)(file[offset] | mask) : (byte)(file[offset] & ~mask);
        }

        public void UnlockAvailableClasses(int recordIndex)
        {
            CheckIndex(recordIndex, CharacterCount, nameof(recordIndex));
            if (!ClassEligibility.IsPlayableRecord(recordIndex))
                throw new ArgumentException($"Record {recordIndex} is not a playable character.", nameof(recordIndex));
            for (int classId = 0; classId < ClassCount; classId++)
                if (ClassEligibility.IsAvailable(recordIndex, classId))
                    SetClassMastered(recordIndex, classId, true);
        }

        public int UnlockAllPlayableSkillsAndClasses()
        {
            int updated = 0;
            for (int record = 0; record < CharacterCount; record++)
            {
                if (!ClassEligibility.IsPlayableRecord(record)) continue;
                for (int skill = 0; skill < SkillCount; skill++)
                {
                    if (GetSkillRank(record, skill) >= MaxSkillRank) continue;
                    SetSkillRank(record, skill, MaxSkillRank);
                    updated++;
                }
                for (int classId = 0; classId < ClassCount; classId++)
                {
                    if (!ClassEligibility.IsAvailable(record, classId)
                        || IsClassMastered(record, classId)) continue;
                    SetClassMastered(record, classId, true);
                    updated++;
                }
            }
            return updated;
        }

        private (int offset, byte mask) ClassBit(int recordIndex, int classId)
        {
            CheckIndex(recordIndex, CharacterCount, nameof(recordIndex));
            CheckIndex(classId, ClassCount, nameof(classId));
            return (player + ClassOffset + recordIndex * ClassBytesPerCharacter + classId / 8,
                (byte)(1 << (classId % 8)));
        }

        public object Snapshot()
        {
            var supports = Enumerable.Range(0, Player_V23.COUNT_SUPPORT)
                .Select(index => new
                {
                    index,
                    name = GetSupportName(index),
                    maxPoints = GetSupportPoints(index),
                    maximumRank = SupportPairRanks.MaxRank(index)
                }).ToArray();
            var characters = Enumerable.Range(0, CharacterCount)
                .Select(index => new
                {
                    recordIndex = index,
                    name = GetCharacterName(index),
                    skillRanks = Enumerable.Range(0, SkillCount).Select(skill => GetSkillRank(index, skill)).ToArray(),
                    masteredClassIds = Enumerable.Range(0, ClassCount)
                        .Where(classId => IsClassMastered(index, classId)).ToArray()
                }).ToArray();
            return new
            {
                professorRank = ProfessorRank,
                supports,
                characters,
                note = "NG+ history is separate from current-run progress. Test changes on a copy before using the save in-game."
            };
        }

        private static string CharacterName(int index)
        {
            if (Database.BinaryDatabase == null)
                return "Record " + index;
            var entries = Database.BinaryDatabase.CharacterEntries;
            string name = Enumerable.Range(0, entries.Count)
                .Where(id => entries[id].MainCharacterId == index)
                .OrderByDescending(id => id >= 1000)
                .Select(id => Database.GetUnitName(id))
                .FirstOrDefault();
            return string.IsNullOrWhiteSpace(name) ? "Record " + index : name;
        }

        private static string SupportName(int index)
        {
            if (Database.BinaryDatabase == null || index >= Database.BinaryDatabase.SupportTalkEntries.Count)
                return "Support " + index;
            return Database.GetSupportTalkName(index);
        }

        private static void CheckIndex(int index, int count, string name)
        {
            if (index < 0 || index >= count) throw new ArgumentOutOfRangeException(name);
        }
    }
}
