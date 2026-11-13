using System;
using System.Collections.Generic;
using System.IO;
using System.Linq;
using System.Reflection;
using System.Runtime.InteropServices;
using System.Security.Cryptography;
using System.Text;
using System.Text.RegularExpressions;
using SaveEditor;
using SaveEditor.Structs;

namespace FethEditor.Core
{
    public sealed partial class SaveBuffer
    {
        private const int HeaderSize = 12;
        private static readonly Regex SegmentPattern = new Regex(@"^([A-Za-z_][A-Za-z_0-9]*)(?:\[(\d+)\])?$", RegexOptions.Compiled);
        private static readonly Regex[] WritablePaths =
        {
            Pattern(@"^items\[\d+\]\.(id|durability|amount)$"),
            Pattern(@"^characters\[\d+\]\.data\.(id|rng_value|exp|level|hp|strength|magic|dexterity|speed|luck|defense|resistance|movement|charm|motivation|currentclassexp|currentclasslevel|adjutantid|flags|classflags)$"),
            Pattern(@"^characters\[\d+\]\.data\.(skillexp|classexp|classlevel|equippedabilities|equippedcombatarts|combatarts|abilities|classunlockflags)\[\d+\]$"),
            Pattern(@"^characters\[\d+\]\.data\.items\[\d+\]\.(id|durability|amount)$"),
            Pattern(@"^player\.(playtime|money|chapter|difficulty|gamestyle|route|mapid)$"),
            Pattern(@"^player\.(charactersupportvalues|miscitems|giftitems|giftitems2)\[\d+\]$"),
            Pattern(@"^player\.battalions\[\d+\]\.(characterid|exp|stamina|type|skill)$"),
            Pattern(@"^activities\.(reputation|instructexp|activityexplore|activitylesson|activitybattle|statue[1-4]|playlog_[a-z]+)$"),
            Pattern(@"^activities\.queststatelist\[\d+\]$")
        };
        private static readonly Regex[] WritableBits =
        {
            Pattern(@"^characters\[\d+\]\.data\.(combatarts|abilities|classunlockflags|flags|classflags)$")
        };

        private readonly byte[] bytes;
        private readonly byte[] original;

        private SaveBuffer(byte[] file, bool hasInvalidChecksum)
        {
            bytes = file;
            original = (byte[])file.Clone();
            HasInvalidChecksum = hasInvalidChecksum;
        }

        public bool HasInvalidChecksum { get; }

        public static SaveBuffer Open(string path)
        {
            byte[] file = File.ReadAllBytes(path);
            if (file.Length != Save.SIZE_SAVE_V23)
                throw new InvalidDataException($"Expected a 1.2.0 slot/auto file of {Save.SIZE_SAVE_V23} bytes; got {file.Length}. Suspend and system files cannot be edited here.");
            if (BitConverter.ToUInt32(file, 4) != Save.CURRENT_VERSION || BitConverter.ToUInt32(file, 8) != file.Length)
                throw new InvalidDataException("Unsupported save version or declared size.");
            bool hasInvalidChecksum = BitConverter.ToUInt32(file, 0) != Checksum(file);
            return new SaveBuffer(file, hasInvalidChecksum);
        }

        public SaveData_V23 Data => Util.ReadStructure<SaveData_V23>(bytes.Skip(HeaderSize).ToArray());
        public string Sha256 => Digest(original);
        public int ChangedBytes => Enumerable.Range(HeaderSize, bytes.Length - HeaderSize).Count(i => bytes[i] != original[i]);

        public object ReadInheritance()
        {
            return Inheritance.Snapshot();
        }

        public NgPlusJournal Inheritance => new NgPlusJournal(bytes, Resolve("Player").Offset);

        public void ReachMaxSupportRank(int index)
        {
            int target = SupportPairRanks.MaxRankPoints(index);
            string path = $"Player.CharacterSupportValues[{index}]";
            Set(path, Math.Max((long)Get(path), target));
        }

        public int ReachMaxSupportRanks()
        {
            int updated = 0;
            for (int index = 0; index < Player_V23.COUNT_SUPPORT; index++)
            {
                if (SupportPairRanks.MaxRank(index) == "None") continue;
                string path = $"Player.CharacterSupportValues[{index}]";
                if ((long)Get(path) >= SupportPairRanks.MaxRankPoints(index)) continue;
                ReachMaxSupportRank(index);
                updated++;
            }
            return updated;
        }

        public object Get(string path)
        {
            if (path.Equals("playerName", StringComparison.OrdinalIgnoreCase))
                return Util.DecodeString(Data.PlayerName);
            if (TryGift(path, out int gift))
                return gift < Player_V23.COUNT_GIFT_ITEMS1
                    ? Get($"player.GiftItems[{gift}]")
                    : Get($"player.GiftItems2[{gift - Player_V23.COUNT_GIFT_ITEMS1}]");
            Location loc = Resolve(path);
            return ReadNumber(loc);
        }

        public void Set(string path, long value)
        {
            if (TryGift(path, out int gift))
            {
                Set(gift < Player_V23.COUNT_GIFT_ITEMS1
                    ? $"player.GiftItems[{gift}]"
                    : $"player.GiftItems2[{gift - Player_V23.COUNT_GIFT_ITEMS1}]", value);
                return;
            }
            if (!WritablePaths.Any(pattern => pattern.IsMatch(path)))
                throw new ArgumentException($"Field is not writable: {path}");
            Location loc = Resolve(path);
            Validate(path, value);
            WriteNumber(loc, value);

            Match skill = Regex.Match(path, @"^(characters\[\d+\]\.data\.)skillexp(\[\d+\])$", RegexOptions.IgnoreCase);
            if (skill.Success)
                WriteNumber(Resolve(skill.Groups[1].Value + "SkillExp2" + skill.Groups[2].Value), value);

            Match current = Regex.Match(path, @"^(characters\[\d+\]\.data\.)currentclass(exp|level)$", RegexOptions.IgnoreCase);
            if (current.Success)
            {
                int classId = checked((int)(long)Get(current.Groups[1].Value + "Class"));
                string field = current.Groups[2].Value.Equals("exp", StringComparison.OrdinalIgnoreCase) ? "ClassExp" : "ClassLevel";
                WriteNumber(Resolve(current.Groups[1].Value + field + "[" + classId + "]"), value);
            }

            Match classValue = Regex.Match(path, @"^(characters\[\d+\]\.data\.)class(exp|level)\[(\d+)\]$", RegexOptions.IgnoreCase);
            if (classValue.Success)
            {
                int classId = checked((int)(long)Get(classValue.Groups[1].Value + "Class"));
                if (classId == int.Parse(classValue.Groups[3].Value))
                    WriteNumber(Resolve(classValue.Groups[1].Value + "CurrentClass" + classValue.Groups[2].Value), value);
            }
            if (Regex.IsMatch(path, @"^items\[\d+\]\.id$", RegexOptions.IgnoreCase))
                RecountItems();
        }

        public void SetName(string name)
        {
            byte[] encoded = Util.MakeName(name, 0x28);
            int offset = HeaderSize + checked((int)Marshal.OffsetOf(typeof(SaveData_V23), "PlayerName"));
            Buffer.BlockCopy(encoded, 0, bytes, offset, encoded.Length);
        }

        public void SetBit(string path, int index, bool enabled)
        {
            if (!WritableBits.Any(pattern => pattern.IsMatch(path)))
                throw new ArgumentException($"Flag group is not writable: {path}");
            Location loc = Resolve(path);
            int byteCount;
            if (loc.Type.IsArray)
                byteCount = loc.ArrayLength * ElementSize(loc.Type.GetElementType());
            else
                byteCount = ElementSize(loc.Type);
            if (index < 0 || index >= byteCount * 8)
                throw new ArgumentOutOfRangeException(nameof(index));
            int offset = loc.Offset + index / 8;
            int mask = 1 << (index % 8);
            bytes[offset] = enabled ? (byte)(bytes[offset] | mask) : (byte)(bytes[offset] & ~mask);
        }

        public byte[] ExportCharacter(int slot)
        {
            Location loc = Resolve($"Characters[{slot}]");
            return bytes.Skip(loc.Offset).Take(Character_V23.SIZE).ToArray();
        }

        public void ImportCharacter(int slot, byte[] character)
        {
            if (character.Length != Character_V23.SIZE)
                throw new InvalidDataException($"Character export must be {Character_V23.SIZE} bytes.");
            Location loc = Resolve($"Characters[{slot}]");
            Buffer.BlockCopy(character, 0, bytes, loc.Offset, character.Length);
        }

        public void RecountItems()
        {
            int count = 0;
            for (int i = 0; i < SaveData_V23.ITEM_COUNT; i++)
                if ((long)Get($"Items[{i}].Id") != -1) count++;
            WriteNumber(Resolve("ItemCount"), count);
        }

        public void SortItems()
        {
            Location baseLoc = Resolve("Items[0]");
            var entries = Enumerable.Range(0, SaveData_V23.ITEM_COUNT)
                .Select(i => bytes.Skip(baseLoc.Offset + 4 * i).Take(4).ToArray())
                .OrderBy(entry => BitConverter.ToInt16(entry, 0) == -1 ? 1 : 0)
                .ThenBy(entry => BitConverter.ToInt16(entry, 0))
                .ThenBy(entry => entry[2]).ToArray();
            for (int i = 0; i < entries.Length; i++)
                Buffer.BlockCopy(entries[i], 0, bytes, baseLoc.Offset + 4 * i, 4);
            RecountItems();
        }

        public void SortBattalions()
        {
            Location baseLoc = Resolve("Player.Battalions[0]");
            var entries = Enumerable.Range(0, Player_V23.COUNT_BATTALION)
                .Select(i => bytes.Skip(baseLoc.Offset + 8 * i).Take(8).ToArray())
                .OrderBy(entry => entry[6] == Database.BATTALION_COUNT ? 1 : 0)
                .ThenBy(entry => entry[6])
                .ThenBy(entry => BitConverter.ToInt16(entry, 0)).ToArray();
            for (int i = 0; i < entries.Length; i++)
                Buffer.BlockCopy(entries[i], 0, bytes, baseLoc.Offset + 8 * i, 8);
        }

        public int FillMissingBattalions()
        {
            SaveData_V23 data = Data;
            var ownedTypes = new HashSet<int>(data.Player.Battalions
                .Where(battalion => battalion.Type < Database.BATTALION_COUNT)
                .Select(battalion => (int)battalion.Type));
            foreach (Character_V23 character in data.Characters)
                if (character.data.Id >= 0 && character.data.Level > 0
                    && character.data.EquippedBattalion.Type < Database.BATTALION_COUNT)
                    ownedTypes.Add(character.data.EquippedBattalion.Type);

            var missing = ObtainableBattalions.All
                .Where(template => !ownedTypes.Contains(template.Type)).ToArray();
            int[] freeSlots = data.Player.Battalions
                .Select((battalion, index) => (battalion, index))
                .Where(entry => entry.battalion.Type == Database.BATTALION_COUNT)
                .Select(entry => entry.index).ToArray();
            if (missing.Length > freeSlots.Length)
                throw new InvalidOperationException(
                    $"Need {missing.Length} free battalion slots, but only {freeSlots.Length} remain.");

            for (int i = 0; i < missing.Length; i++)
            {
                string path = $"Player.Battalions[{freeSlots[i]}].";
                Set(path + "CharacterId", -1);
                Set(path + "Exp", ObtainableBattalions.Experience);
                Set(path + "Stamina", missing[i].Stamina);
                Set(path + "Type", missing[i].Type);
                Set(path + "Skill", missing[i].Skill);
            }
            if (missing.Length > 0) SortBattalions();
            return missing.Length;
        }

        public void SetInventoryDurability(string mode)
        {
            if (mode != "normal" && mode != "unlimited" && mode != "weapons-unlimited")
                throw new ArgumentException("Durability mode must be normal, unlimited, or weapons-unlimited.");
            for (int i = 0; i < SaveData_V23.ITEM_COUNT; i++)
            {
                short id = checked((short)(long)Get($"Items[{i}].Id"));
                if (id == -1) continue;
                int durability = mode == "unlimited" || (mode == "weapons-unlimited" && id >= 10 && id < 510)
                    ? 100 : Database.GetItemDurability(id);
                Set($"Items[{i}].Durability", durability);
            }
        }

        public void RestoreCharacterItemDurability(int slot)
        {
            int count = checked((int)(long)Get($"Characters[{slot}].data.ItemCount"));
            if (count > Database.MAX_CHARA_ITEMS)
                throw new InvalidDataException("Character item count is invalid.");
            var items = new List<(int Index, byte Durability)>();
            for (int i = 0; i < count; i++)
            {
                int id = checked((int)(long)Get($"Characters[{slot}].data.Items[{i}].Id"));
                if (id == -1) continue;
                if (!Database.ItemList.ContainsKey(id))
                    throw new InvalidDataException($"Unknown character item ID {id}.");
                items.Add((i, checked((byte)Database.GetItemDurability(id))));
            }
            foreach (var item in items)
                Set($"Characters[{slot}].data.Items[{item.Index}].Durability", item.Durability);
        }

        public void SetCharacterItem(int slot, int itemSlot, short id, byte durability)
        {
            if (slot < 0 || slot >= SaveData_V23.CHARACTER_COUNT)
                throw new ArgumentOutOfRangeException(nameof(slot));
            if (itemSlot < 0 || itemSlot >= Database.MAX_CHARA_ITEMS)
                throw new ArgumentOutOfRangeException(nameof(itemSlot));
            if (id == -1 || !Database.ItemList.ContainsKey(id))
                throw new ArgumentException("Choose an item from the catalog.", nameof(id));
            int maximumDurability = Database.GetItemDurability(id);
            if (durability > maximumDurability)
                throw new ArgumentOutOfRangeException(nameof(durability),
                    $"Item durability cannot exceed {maximumDurability}.");

            string prefix = $"Characters[{slot}].data";
            int count = checked((int)(long)Get($"{prefix}.ItemCount"));
            if (count > Database.MAX_CHARA_ITEMS)
                throw new InvalidDataException("Character item count is invalid.");
            for (int index = 0; index < Database.MAX_CHARA_ITEMS; index++)
            {
                bool occupied = (long)Get($"{prefix}.Items[{index}].Id") != -1;
                if (occupied != (index < count))
                    throw new InvalidDataException("Character item slots do not match their item count.");
            }
            if (itemSlot > count)
                throw new InvalidOperationException("Add items to the first empty slot.");

            string item = $"{prefix}.Items[{itemSlot}]";
            Set($"{item}.Id", id);
            Set($"{item}.Durability", durability);
            if (itemSlot == count)
                WriteNumber(Resolve($"{prefix}.ItemCount"), count + 1);
        }

        public void MaxSkillExperience(int slot)
        {
            for (int i = 0; i < Database.MAX_SKILLS; i++)
            {
                int rank = checked((int)(long)Get($"Characters[{slot}].data.SkillLevel[{i}]"));
                if (rank < 0 || rank >= Database.SkillLevelupRank.Length)
                    throw new InvalidDataException("Character skill rank is outside the game's table.");
                Set($"Characters[{slot}].data.SkillExp[{i}]", Database.SkillLevelupRank[rank] - 1);
            }
        }

        public void SetSkillRank(int slot, int skill, int rank, int experience)
        {
            if (skill < 0 || skill >= Database.MAX_SKILLS)
                throw new ArgumentOutOfRangeException(nameof(skill));
            if (rank < 0 || rank >= Database.SkillLevelupRank.Length)
                throw new ArgumentOutOfRangeException(nameof(rank));
            if (experience < 0 || experience >= Database.SkillLevelupRank[rank])
                throw new ArgumentOutOfRangeException(nameof(experience));

            string prefix = $"Characters[{slot}].data.";
            Location level = Resolve(prefix + $"SkillLevel[{skill}]");
            Location mirror = Resolve(prefix + $"SkillLevel2[{skill}]");
            WriteNumber(level, rank);
            WriteNumber(mirror, rank);
            Set(prefix + $"SkillExp[{skill}]", experience);
        }

        public void MaxClassExperience(int slot)
        {
            int unitId = Data.Characters[slot].data.Id;
            int record = ClassEligibility.RecordForUnit(unitId);
            if (record < 0)
                throw new InvalidOperationException($"No playable character rules are known for unit {unitId}.");
            for (int i = 0; i < Database.MAX_CLASS; i++)
                if (ClassEligibility.IsAvailable(record, i))
                    Set($"Characters[{slot}].data.ClassExp[{i}]", Database.GetMaxClassExp(i));
        }

        public void UnlockAll(int slot, string kind)
        {
            string path;
            int count;
            if (kind == "abilities")
            {
                path = $"Characters[{slot}].data.Abilities";
                count = Database.MAX_ABILITIES * 8;
            }
            else if (kind == "combat-arts")
            {
                path = $"Characters[{slot}].data.CombatArts";
                count = Database.COMBAT_ARTS_COUNT;
            }
            else
                throw new ArgumentException("Unlock kind must be abilities or combat-arts.");
            for (int i = 0; i < count; i++) SetBit(path, i, true);
        }

        public void FillItems(string kind, byte amount)
        {
            if (kind == "misc")
                for (int i = 0; i < Player_V23.COUNT_MISC_ITEMS; i++) Set($"Player.MiscItems[{i}]", amount);
            else if (kind == "gifts")
                for (int i = 0; i < Player_V23.COUNT_GIFT_ITEMS; i++) Set($"gifts[{i}]", amount);
            else
                throw new ArgumentException("Fill kind must be misc or gifts.");
        }

        public void AddEssentialItems()
        {
            foreach (short id in Database.EssentialItems.Distinct())
            {
                int slot = -1;
                int free = -1;
                for (int i = 0; i < SaveData_V23.ITEM_COUNT; i++)
                {
                    int itemId = checked((int)(long)Get($"Items[{i}].Id"));
                    if (itemId == id) { slot = i; break; }
                    if (itemId == -1 && free == -1) free = i;
                }
                if (slot == -1) slot = free;
                if (slot == -1) throw new InvalidOperationException("Inventory has no free slot for essential items.");
                Set($"Items[{slot}].Id", id);
                Set($"Items[{slot}].Durability", Database.GetItemDurability(id));
                Set($"Items[{slot}].Amount", 99);
            }
            SortItems();
        }

        public IReadOnlyList<object> Differences()
        {
            var result = new List<object>();
            for (int i = HeaderSize; i < bytes.Length; i++)
                if (bytes[i] != original[i])
                    result.Add(new { offset = i, before = original[i], after = bytes[i] });
            return result;
        }

        public byte[] FinishedBytes()
        {
            byte[] result = (byte[])bytes.Clone();
            byte[] sum = BitConverter.GetBytes(Checksum(result));
            Buffer.BlockCopy(sum, 0, result, 0, 4);
            return result;
        }

        public static string Digest(byte[] data)
        {
            using (var sha = SHA256.Create())
                return BitConverter.ToString(sha.ComputeHash(data)).Replace("-", "").ToLowerInvariant();
        }

        private static Regex Pattern(string value) => new Regex(value, RegexOptions.IgnoreCase | RegexOptions.Compiled);

        private static bool TryGift(string path, out int index)
        {
            Match match = Regex.Match(path, @"^gifts\[(\d+)\]$", RegexOptions.IgnoreCase);
            index = match.Success ? int.Parse(match.Groups[1].Value) : -1;
            if (match.Success && index >= Player_V23.COUNT_GIFT_ITEMS)
                throw new ArgumentOutOfRangeException(nameof(path));
            return match.Success;
        }

        private static uint Checksum(byte[] file)
        {
            uint sum = 0;
            for (int i = HeaderSize; i < file.Length; i++)
                unchecked { sum += file[i]; }
            return sum;
        }

        private static int ElementSize(Type type) => type.IsPrimitive ? Marshal.SizeOf(type) : Marshal.SizeOf(type);

        private Location Resolve(string path)
        {
            Type type = typeof(SaveData_V23);
            int offset = HeaderSize;
            foreach (string part in path.Split('.'))
            {
                Match segment = SegmentPattern.Match(part);
                if (!segment.Success)
                    throw new ArgumentException($"Invalid field path: {path}");
                FieldInfo field = type.GetField(segment.Groups[1].Value, BindingFlags.Public | BindingFlags.Instance | BindingFlags.IgnoreCase);
                if (field == null)
                    throw new ArgumentException($"Unknown field: {path}");
                offset += checked((int)Marshal.OffsetOf(type, field.Name));
                type = field.FieldType;
                if (segment.Groups[2].Success)
                {
                    if (!type.IsArray)
                        throw new ArgumentException($"Field is not an array: {path}");
                    int length = field.GetCustomAttribute<MarshalAsAttribute>().SizeConst;
                    int index = int.Parse(segment.Groups[2].Value);
                    if (index < 0 || index >= length)
                        throw new ArgumentOutOfRangeException(nameof(path), $"Index {index} is outside {field.Name}[{length}].");
                    type = type.GetElementType();
                    offset += checked(index * ElementSize(type));
                }
            }
            int arrayLength = 0;
            if (type.IsArray)
            {
                // The last segment's MarshalAs length is required for bit operations.
                string last = path.Split('.').Last();
                string parent = path.Substring(0, path.Length - last.Length).TrimEnd('.');
                Type parentType = parent.Length == 0 ? typeof(SaveData_V23) : Resolve(parent).Type;
                FieldInfo field = parentType.GetField(last, BindingFlags.Public | BindingFlags.Instance | BindingFlags.IgnoreCase);
                arrayLength = field.GetCustomAttribute<MarshalAsAttribute>().SizeConst;
            }
            return new Location(offset, type, arrayLength);
        }

        private long ReadNumber(Location loc)
        {
            if (loc.Type == typeof(byte)) return bytes[loc.Offset];
            if (loc.Type == typeof(short)) return BitConverter.ToInt16(bytes, loc.Offset);
            if (loc.Type == typeof(ushort)) return BitConverter.ToUInt16(bytes, loc.Offset);
            if (loc.Type == typeof(int)) return BitConverter.ToInt32(bytes, loc.Offset);
            if (loc.Type == typeof(uint)) return BitConverter.ToUInt32(bytes, loc.Offset);
            throw new ArgumentException("The path must point to a numeric field.");
        }

        private void WriteNumber(Location loc, long value)
        {
            byte[] encoded;
            if (loc.Type == typeof(byte)) encoded = new[] { checked((byte)value) };
            else if (loc.Type == typeof(short)) encoded = BitConverter.GetBytes(checked((short)value));
            else if (loc.Type == typeof(ushort)) encoded = BitConverter.GetBytes(checked((ushort)value));
            else if (loc.Type == typeof(int)) encoded = BitConverter.GetBytes(checked((int)value));
            else if (loc.Type == typeof(uint)) encoded = BitConverter.GetBytes(checked((uint)value));
            else throw new ArgumentException("The path must point to a numeric field.");
            Buffer.BlockCopy(encoded, 0, bytes, loc.Offset, encoded.Length);
        }

        private void Validate(string path, long value)
        {
            if (Regex.IsMatch(path, @"^player\.playtime$", RegexOptions.IgnoreCase) && (value < 0 || value > Database.MAX_PLAYTIME))
                throw new ArgumentOutOfRangeException(nameof(value), "Playtime exceeds the GUI limit.");
            if (Regex.IsMatch(path, @"^player\.money$", RegexOptions.IgnoreCase) && (value < 0 || value > Database.MAX_MONEY))
                throw new ArgumentOutOfRangeException(nameof(value), "Money exceeds the GUI limit.");
            if (Regex.IsMatch(path, @"^player\.difficulty$", RegexOptions.IgnoreCase) && (value < 0 || value >= Database.DIFFICULTY_COUNT))
                throw new ArgumentOutOfRangeException(nameof(value), "Difficulty is outside the GUI list.");
            if (Regex.IsMatch(path, @"^player\.gamestyle$", RegexOptions.IgnoreCase) && (value < 0 || value >= Database.GAMESTYLE_COUNT))
                throw new ArgumentOutOfRangeException(nameof(value), "Game style is outside the GUI list.");
            if (Regex.IsMatch(path, @"^activities\.reputation$", RegexOptions.IgnoreCase) && (value < 0 || value > Database.MAX_REPUTATION))
                throw new ArgumentOutOfRangeException(nameof(value), "Renown exceeds the GUI limit.");
            if (Regex.IsMatch(path, @"^activities\.instructexp$", RegexOptions.IgnoreCase) && (value < 0 || value > Database.MAX_INSTRUCT_EXP))
                throw new ArgumentOutOfRangeException(nameof(value), "Professor experience exceeds the GUI limit.");
            if (Regex.IsMatch(path, @"\.classlevel\[\d+\]$|\.currentclasslevel$", RegexOptions.IgnoreCase) && (value < 0 || value > Database.MAX_CLASS_LEVEL))
                throw new ArgumentOutOfRangeException(nameof(value), "Class mastery must be 0 or 1.");
            Match classExp = Regex.Match(path, @"^(characters\[\d+\]\.data\.)classexp\[(\d+)\]$", RegexOptions.IgnoreCase);
            if (classExp.Success && (value < 0 || value > Database.GetMaxClassExp(int.Parse(classExp.Groups[2].Value))))
                throw new ArgumentOutOfRangeException(nameof(value), "Class experience exceeds the GUI limit.");
            Match currentClassExp = Regex.Match(path, @"^(characters\[\d+\]\.data\.)currentclassexp$", RegexOptions.IgnoreCase);
            if (currentClassExp.Success)
            {
                int classId = checked((int)(long)Get(currentClassExp.Groups[1].Value + "Class"));
                if (value < 0 || value > Database.GetMaxClassExp(classId))
                    throw new ArgumentOutOfRangeException(nameof(value), "Current class experience exceeds the GUI limit.");
            }
            Match skillExp = Regex.Match(path, @"^(characters\[\d+\]\.data\.)skillexp\[(\d+)\]$", RegexOptions.IgnoreCase);
            if (skillExp.Success)
            {
                int rank = checked((int)(long)Get(skillExp.Groups[1].Value + "SkillLevel[" + skillExp.Groups[2].Value + "]"));
                if (rank >= Database.SkillLevelupRank.Length || value < 0 || value >= Database.SkillLevelupRank[rank])
                    throw new ArgumentOutOfRangeException(nameof(value), "Skill experience exceeds the current rank limit in the GUI.");
            }
            if (Regex.IsMatch(path, @"^activities\.queststatelist\[\d+\]$", RegexOptions.IgnoreCase) && (value < 0 || value > 6))
                throw new ArgumentOutOfRangeException(nameof(value), "Quest state must be 0 through 6.");
            Match characterStat = Regex.Match(path,
                @"^characters\[(\d+)\]\.data\.(hp|strength|magic|dexterity|speed|luck|defense|resistance|movement|charm)$",
                RegexOptions.IgnoreCase);
            if (characterStat.Success)
            {
                int slot = int.Parse(characterStat.Groups[1].Value);
                var character = Data.Characters[slot].data;
                string stat = characterStat.Groups[2].Value.ToLowerInvariant();
                int statIndex = Array.IndexOf(new[]
                {
                    "strength", "magic", "dexterity", "speed", "luck", "defense",
                    "resistance", "movement", "charm"
                }, stat);
                int maximum = stat == "hp"
                    ? Database.GetMaxHP(character.Id, character.Class)
                    : Database.GetMaxStat(character.Id, character.Class, statIndex);
                if (value < 0 || value > maximum)
                    throw new ArgumentOutOfRangeException(nameof(value),
                        $"{characterStat.Groups[2].Value} exceeds this character's maximum of {maximum}.");
            }
        }

        private sealed class Location
        {
            public Location(int offset, Type type, int arrayLength)
            {
                Offset = offset;
                Type = type;
                ArrayLength = arrayLength;
            }
            public int Offset { get; }
            public Type Type { get; }
            public int ArrayLength { get; }
        }
    }
}
