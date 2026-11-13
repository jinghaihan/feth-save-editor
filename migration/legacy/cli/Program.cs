using System;
using System.Collections;
using System.Collections.Generic;
using System.Globalization;
using System.IO;
using System.Linq;
#if MODERN_DOTNET
using JavaScriptSerializer = FethEditor.Cli.JsonCompat;
#else
using System.Web.Script.Serialization;
#endif
using FethEditor.Core;
using SaveEditor;
using SaveEditor.Structs;

namespace FethEditor.Cli
{
    internal static class Program
    {
        private static readonly JavaScriptSerializer Json = new JavaScriptSerializer { MaxJsonLength = 32 * 1024 * 1024, RecursionLimit = 100 };

        private static int Main(string[] args)
        {
            try
            {
                if (args.Length == 0 || args[0] == "help" || args[0] == "--help")
                {
                    Console.WriteLine("FethEditor.Cli inspect|get|apply|inspect-system|apply-system|catalog|export-character|import-character --input <save> [options]");
                    return 0;
                }

                string command = args[0].ToLowerInvariant();
                Dictionary<string, string> options = ParseOptions(args.Skip(1).ToArray());
                enmLanguage language = enmLanguage.en_u;
                if (options.TryGetValue("language", out string lang))
                    language = (enmLanguage)Enum.Parse(typeof(enmLanguage), lang, true);
                Database.Init(language);

                if (command == "catalog")
                {
                    string type = Required(options, "type");
                    Print(options.TryGetValue("id", out string id)
                        ? CatalogDetail(type, int.Parse(id, CultureInfo.InvariantCulture))
                        : Catalog(type));
                    return 0;
                }

                string input = Required(options, "input");
                if (command == "inspect-system" || command == "apply-system")
                {
                    SystemBuffer system = SystemBuffer.Open(input);
                    if (command == "inspect-system") Print(InspectSystem(system, input,
                        options.TryGetValue("section", out string systemSection) ? systemSection : "all"));
                    else ApplySystem(system, input, options);
                    return 0;
                }
                SaveBuffer save = SaveBuffer.Open(input);

                switch (command)
                {
                    case "inspect":
                        Print(Inspect(save, options.TryGetValue("section", out string section) ? section : "all"));
                        break;
                    case "get":
                        string path = Required(options, "path");
                        Print(new { path, value = save.Get(path), sha256 = save.Sha256 });
                        break;
                    case "apply":
                        Apply(save, input, options);
                        break;
                    case "export-character":
                        int exportSlot = int.Parse(Required(options, "slot"), CultureInfo.InvariantCulture);
                        string exportPath = Required(options, "output");
                        if (File.Exists(exportPath)) throw new IOException("Output already exists: " + exportPath);
                        File.WriteAllBytes(exportPath, save.ExportCharacter(exportSlot));
                        Print(new { output = Path.GetFullPath(exportPath), slot = exportSlot, bytes = Character_V23.SIZE });
                        break;
                    case "import-character":
                        var importPatch = new Dictionary<string, object>
                        {
                            { "expectedSha256", save.Sha256 },
                            { "operations", new object[] { new Dictionary<string, object>
                                {
                                    { "op", "importCharacter" },
                                    { "slot", int.Parse(Required(options, "slot"), CultureInfo.InvariantCulture) },
                                    { "file", Required(options, "character") }
                                } } }
                        };
                        ApplyOperations(save, importPatch);
                        Finish(save, input, options);
                        break;
                    default:
                        throw new ArgumentException("Unknown command: " + command);
                }
                return 0;
            }
            catch (Exception error)
            {
                Console.Error.WriteLine(Json.Serialize(new { error = error.Message, type = error.GetType().Name }));
                return 1;
            }
        }

        private static Dictionary<string, string> ParseOptions(string[] args)
        {
            var result = new Dictionary<string, string>(StringComparer.OrdinalIgnoreCase);
            for (int i = 0; i < args.Length; i++)
            {
                if (!args[i].StartsWith("--", StringComparison.Ordinal))
                    throw new ArgumentException("Expected an --option, got " + args[i]);
                string key = args[i].Substring(2);
                if (key == "in-place" || key == "dry-run")
                    result[key] = "true";
                else
                {
                    if (++i == args.Length) throw new ArgumentException("Missing value for --" + key);
                    result[key] = args[i];
                }
            }
            return result;
        }

        private static string Required(Dictionary<string, string> options, string key)
        {
            if (!options.TryGetValue(key, out string value) || string.IsNullOrWhiteSpace(value))
                throw new ArgumentException("Missing --" + key);
            return value;
        }

        private static void Apply(SaveBuffer save, string input, Dictionary<string, string> options)
        {
            var patch = ReadPatch(options);
            ApplyOperations(save, patch);
            Finish(save, input, options);
        }

        private static Dictionary<string, object> ReadPatch(Dictionary<string, string> options) =>
            Json.DeserializeObject(File.ReadAllText(Required(options, "patch"))) as Dictionary<string, object>
            ?? throw new InvalidDataException("Patch must be a JSON object.");

        private static void ApplyOperations(SaveBuffer save, Dictionary<string, object> patch)
        {
            if (patch.TryGetValue("expectedSha256", out object expected)
                && !string.Equals(Convert.ToString(expected), save.Sha256, StringComparison.OrdinalIgnoreCase))
                throw new InvalidDataException("Input SHA-256 differs from patch expectation.");
            if (!patch.TryGetValue("operations", out object operations) || !(operations is object[] list))
                throw new InvalidDataException("Patch needs an operations array.");
            foreach (object operation in list)
            {
                var item = operation as Dictionary<string, object>;
                if (item == null) throw new InvalidDataException("Every operation must be an object.");
                string op = Value<string>(item, "op");
                switch (op)
                {
                    case "set":
                        save.Set(Value<string>(item, "path"), Value<long>(item, "value"));
                        break;
                    case "setName":
                        save.SetName(Value<string>(item, "value"));
                        break;
                    case "setProtagonistGender":
                        save.SetProtagonistGender(Value<string>(item, "gender").ToLowerInvariant() switch
                        {
                            "male" => 0,
                            "female" => 1,
                            _ => throw new ArgumentException("Gender must be male or female.")
                        });
                        break;
                    case "setBit":
                        save.SetBit(Value<string>(item, "path"), Value<int>(item, "index"), Value<bool>(item, "value"));
                        break;
                    case "sortItems":
                        save.SortItems();
                        break;
                    case "sortBattalions":
                        save.SortBattalions();
                        break;
                    case "fillMissingBattalions":
                        save.FillMissingBattalions();
                        break;
                    case "setBattalionEndurance":
                        save.SetBattalionEndurance(Value<int>(item, "slot"), checked((ushort)Value<int>(item, "endurance")));
                        break;
                    case "setBattalionEnduranceValues":
                        save.SetBattalionEnduranceValues(Value<int>(item, "slot"),
                            checked((ushort)Value<int>(item, "storedEndurance")),
                            item.ContainsKey("equippedEndurance")
                                ? checked((ushort)Value<int>(item, "equippedEndurance")) : null);
                        break;
                    case "replenishBattalion":
                        save.ReplenishBattalion(Value<int>(item, "slot"));
                        break;
                    case "replenishCharacterBattalion":
                        save.ReplenishCharacterBattalion(Value<int>(item, "slot"));
                        break;
                    case "replenishBattalions":
                        save.ReplenishBattalions();
                        break;
                    case "maxBattalionLevel":
                        save.MaximizeBattalionLevel(Value<int>(item, "slot"));
                        break;
                    case "maxBattalionLevels":
                        save.MaximizeBattalionLevels();
                        break;
                    case "deleteBattalion":
                        save.DeleteBattalion(Value<int>(item, "slot"));
                        break;
                    case "inventoryDurability":
                        save.SetInventoryDurability(Value<string>(item, "mode"));
                        break;
                    case "characterItemDurability":
                        save.RestoreCharacterItemDurability(Value<int>(item, "slot"));
                        break;
                    case "setCharacterItem":
                        int itemId = Value<int>(item, "id");
                        int durability = item.TryGetValue("durability", out object explicitDurability)
                            ? Convert.ToInt32(explicitDurability, CultureInfo.InvariantCulture)
                            : Database.GetItemDurability(itemId);
                        save.SetCharacterItem(Value<int>(item, "slot"), Value<int>(item, "itemSlot"),
                            checked((short)itemId), checked((byte)durability));
                        break;
                    case "maxSkillExp":
                        save.MaxSkillExperience(Value<int>(item, "slot"));
                        break;
                    case "setSkillRank":
                        save.SetSkillRank(Value<int>(item, "slot"), Value<int>(item, "skill"),
                            Value<int>(item, "rank"), Value<int>(item, "experience"));
                        break;
                    case "setProfessorRank":
                        int professorRank = Value<int>(item, "rank");
                        if (professorRank < 0 || professorRank >= Database.TeacherLevelupRank.Length)
                            throw new ArgumentOutOfRangeException("rank");
                        save.Set("Activities.InstructExp", Database.TeacherLevelupRank[professorRank]);
                        break;
                    case "setSupportRank":
                        int supportIndex = Value<int>(item, "index");
                        save.Set($"Player.CharacterSupportValues[{supportIndex}]",
                            SupportPairRanks.PointsFor(supportIndex, Value<string>(item, "rank")));
                        break;
                    case "maxSupportRank":
                        save.ReachMaxSupportRank(Value<int>(item, "index"));
                        break;
                    case "setNgPlusProfessorRank":
                        save.Inheritance.SetProfessorRank(Value<int>(item, "rank"));
                        break;
                    case "setNgPlusSupport":
                        save.Inheritance.SetSupportPoints(Value<int>(item, "index"), Value<int>(item, "points"));
                        break;
                    case "setNgPlusSupportRank":
                        int inheritedSupportIndex = Value<int>(item, "index");
                        save.Inheritance.SetSupportPoints(inheritedSupportIndex,
                            SupportPairRanks.PointsFor(inheritedSupportIndex, Value<string>(item, "rank")));
                        break;
                    case "maxNgPlusSupportRank":
                        save.Inheritance.ReachMaxSupportRank(Value<int>(item, "index"));
                        break;
                    case "maxNgPlusSupports":
                        save.Inheritance.ReachMaxSupportRanks();
                        break;
                    case "maxSupports":
                        save.ReachMaxSupportRanks();
                        break;
                    case "unlockNgPlusRoster":
                        save.Inheritance.UnlockAllPlayableSkillsAndClasses();
                        break;
                    case "setNgPlusSkillRank":
                        save.Inheritance.SetSkillRank(Value<int>(item, "recordIndex"), Value<int>(item, "skill"),
                            Value<int>(item, "rank"));
                        break;
                    case "setNgPlusClassMastery":
                        save.Inheritance.SetClassMastered(Value<int>(item, "recordIndex"), Value<int>(item, "classId"),
                            Value<bool>(item, "mastered"));
                        break;
                    case "unlockNgPlusClasses":
                        save.Inheritance.UnlockAvailableClasses(Value<int>(item, "recordIndex"));
                        break;
                    case "maxClassExp":
                        save.MaxClassExperience(Value<int>(item, "slot"));
                        break;
                    case "unlockAll":
                        save.UnlockAll(Value<int>(item, "slot"), Value<string>(item, "kind"));
                        break;
                    case "fillItems":
                        save.FillItems(Value<string>(item, "kind"), checked((byte)Value<int>(item, "amount")));
                        break;
                    case "addEssentialItems":
                        save.AddEssentialItems();
                        break;
                    case "importCharacter":
                        save.ImportCharacter(Value<int>(item, "slot"), File.ReadAllBytes(Value<string>(item, "file")));
                        break;
                    default:
                        throw new ArgumentException("Unknown patch operation: " + op);
                }
            }
        }

        private static T Value<T>(Dictionary<string, object> objectValue, string key)
        {
            if (!objectValue.TryGetValue(key, out object value)) throw new ArgumentException("Missing " + key);
            if (typeof(T) == typeof(bool))
            {
                if (!(value is bool)) throw new ArgumentException(key + " must be a boolean.");
                return (T)value;
            }
            if (typeof(T) == typeof(string))
            {
                if (!(value is string)) throw new ArgumentException(key + " must be a string.");
                return (T)value;
            }
            return (T)Convert.ChangeType(value, typeof(T), CultureInfo.InvariantCulture);
        }

        private static object InspectSystem(SystemBuffer system, string input, string section)
        {
            var result = new Dictionary<string, object>
            {
                ["format"] = "FE3H system save",
                ["sha256"] = SaveBuffer.Digest(File.ReadAllBytes(input)),
                ["sourceVersion"] = system.SourceVersion,
                ["checksumValid"] = !system.HasInvalidChecksum
            };
            if (section == "all" || section == "slots")
                result["slots"] = system.Data.Infos.Select((slot, index) => new
                {
                    index, slot.Flags, playerName = slot.GetPlayerName(), slot.Playtime,
                    slot.Chapter1, slot.Chapter2, slot.Day, slot.PlaceId
                }).ToArray();
            if (section == "all" || section == "flags")
                result["flags"] = Enumerable.Range(0, SystemSaveData_V7.COUNT_FLAGS)
                    .Select(index => new { index, enabled = system.GetFlag(index),
                        name = index is >= 8 and < 108 ? SafeName(() => Database.GetString(12823 + index - 8)) : null })
                    .ToArray();
            if (result.Count == 4) throw new ArgumentException("Unknown system section: " + section);
            return result;
        }

        private static void ApplySystem(SystemBuffer system, string input, Dictionary<string, string> options)
        {
            var patch = ReadPatch(options);
            byte[] original = File.ReadAllBytes(input);
            if (patch.TryGetValue("expectedSha256", out object expected)
                && !string.Equals(Convert.ToString(expected), SaveBuffer.Digest(original), StringComparison.OrdinalIgnoreCase))
                throw new InvalidDataException("Input SHA-256 differs from patch expectation.");
            if (!patch.TryGetValue("operations", out object operations) || !(operations is object[] list))
                throw new InvalidDataException("Patch needs an operations array.");

            bool[] before = Enumerable.Range(0, SystemSaveData_V7.COUNT_FLAGS).Select(system.GetFlag).ToArray();
            foreach (object operation in list)
            {
                if (operation is not Dictionary<string, object> item)
                    throw new InvalidDataException("Every operation must be an object.");
                if (Value<string>(item, "op") != "setSystemFlag")
                    throw new ArgumentException("Unknown system patch operation: " + Value<string>(item, "op"));
                system.SetFlag(Value<int>(item, "index"), Value<bool>(item, "value"));
            }
            int[] changedFlags = Enumerable.Range(0, before.Length)
                .Where(index => before[index] != system.GetFlag(index)).ToArray();
            byte[] result = system.FinishedBytes();
            if (options.ContainsKey("dry-run"))
            {
                Print(new { dryRun = true, inputSha256 = SaveBuffer.Digest(original),
                    resultSha256 = SaveBuffer.Digest(result), sourceVersion = system.SourceVersion,
                    outputVersion = SystemSave.CURRENT_VERSION, changedFlags });
                return;
            }
            if (changedFlags.Length == 0)
                throw new InvalidOperationException("Patch changed no system flags; output was not written.");
            string destination = Destination(input, options);
            string backup = VerifiedFileWriter.Write(destination, result, path =>
            {
                var verified = SystemBuffer.Open(path);
                return verified.SourceVersion == SystemSave.CURRENT_VERSION && !verified.HasInvalidChecksum
                    && SaveBuffer.Digest(File.ReadAllBytes(path)) == SaveBuffer.Digest(result);
            });
            Print(new { inputSha256 = SaveBuffer.Digest(original), resultSha256 = SaveBuffer.Digest(result),
                changedFlags, output = destination, backup, outputVersion = SystemSave.CURRENT_VERSION });
        }

        private static string Destination(string input, Dictionary<string, string> options)
        {
            bool inPlace = options.ContainsKey("in-place");
            if (inPlace && options.ContainsKey("output"))
                throw new ArgumentException("Use either --in-place or --output, not both.");
            if (!inPlace && !options.ContainsKey("output"))
                throw new ArgumentException("Provide --output or --in-place.");
            string destination = Path.GetFullPath(inPlace ? input : Required(options, "output"));
            if (!inPlace && Path.GetFullPath(input).Equals(destination, StringComparison.OrdinalIgnoreCase))
                throw new ArgumentException("Use --in-place to modify the input file.");
            if (!inPlace && File.Exists(destination))
                throw new IOException("Output already exists: " + destination);
            return destination;
        }

        private static void Finish(SaveBuffer save, string input, Dictionary<string, string> options)
        {
            bool inPlace = options.ContainsKey("in-place");
            bool dryRun = options.ContainsKey("dry-run");
            if (inPlace && options.ContainsKey("output"))
                throw new ArgumentException("Use either --in-place or --output, not both.");
            if (!inPlace && !dryRun && !options.ContainsKey("output"))
                throw new ArgumentException("Provide --output or --in-place.");

            byte[] result = save.FinishedBytes();
            string resultHash = SaveBuffer.Digest(result);
            if (dryRun)
            {
                Print(new { dryRun = true, inputSha256 = save.Sha256, resultSha256 = resultHash,
                    changedBytes = save.ChangedBytes, differences = save.Differences() });
                return;
            }
            if (save.ChangedBytes == 0)
                throw new InvalidOperationException("Patch changed no data; output was not written.");

            string destination = Path.GetFullPath(inPlace ? input : Required(options, "output"));
            if (!inPlace && Path.GetFullPath(input).Equals(destination, StringComparison.OrdinalIgnoreCase))
                throw new ArgumentException("Use --in-place to modify the input file.");
            if (!inPlace && File.Exists(destination))
                throw new IOException("Output already exists: " + destination);
            string temporary = destination + ".tmp-" + Guid.NewGuid().ToString("N");
            string backup = null;
            try
            {
                File.WriteAllBytes(temporary, result);
                SaveBuffer verified = SaveBuffer.Open(temporary);
                if (verified.Sha256 != resultHash)
                    throw new InvalidDataException("Written save failed verification.");
                if (inPlace)
                {
                    backup = destination + ".backup-" + DateTime.UtcNow.ToString("yyyyMMddTHHmmssfffZ", CultureInfo.InvariantCulture);
                    File.Replace(temporary, destination, backup);
                }
                else
                    File.Move(temporary, destination);
            }
            finally
            {
                if (File.Exists(temporary)) File.Delete(temporary);
            }
            Print(new { inputSha256 = save.Sha256, resultSha256 = resultHash,
                changedBytes = save.ChangedBytes, output = destination, backup });
        }

        private static object Inspect(SaveBuffer save, string section)
        {
            SaveData_V23 data = save.Data;
            var result = new Dictionary<string, object>
            {
                ["format"] = "FE3H 1.1.0-1.2.0 / save version 23",
                ["sha256"] = save.Sha256,
                ["checksumValid"] = !save.HasInvalidChecksum,
                ["note"] = "Mapped current-run fields and separately reported NG+ journal history; other unknown data is preserved."
            };
            if (section == "all" || section == "inheritance")
                result["inheritance"] = save.ReadInheritance();
            if (section == "all" || section == "summary")
                result["summary"] = new
                {
                    playerName = Util.DecodeString(data.PlayerName),
                    protagonistGender = save.ReadProtagonistGender() switch { 0 => "male", 1 => "female", _ => null },
                    data.Player.Playtime, data.Player.Money, data.Player.Chapter,
                    data.Player.Difficulty, data.Player.Gamestyle, data.Player.Route, data.Player.MapID,
                    renown = data.Activities.Reputation, professorExp = data.Activities.InstructExp,
                    data.ItemCount
                };
            if (section == "all" || section == "characters")
                result["characters"] = data.Characters.Select((character, slot) => new
                {
                    slot, id = character.data.Id,
                    name = SafeName(() => Database.GetUnitName(character.data.Id)),
                    character.data.Level, character.data.Exp, character.data.Class,
                    className = SafeName(() => Database.GetClassName(character.data.Class)),
                    character.data.CurrentClassExp, character.data.CurrentClassLevel,
                    character.data.HP, character.data.Strength, character.data.Magic,
                    character.data.Dexterity, character.data.Speed, character.data.Luck,
                    character.data.Defense, character.data.Resistance,
                    character.data.Movement, character.data.Charm, character.data.Motivation,
                    character.data.RNG_VALUE, character.data.AdjutantId,
                    character.data.SkillExp, character.data.SkillLevel,
                    character.data.ClassExp, character.data.ClassLevel,
                    character.data.CombatArts, character.data.Abilities,
                    character.data.EquippedAbilities, character.data.EquippedCombatArts,
                    character.data.Flags, character.data.ClassUnlockFlags, character.data.ClassFlags,
                    character.data.ItemCount,
                    items = character.data.Items.Select((item, index) => ItemInfo(item, index)).ToArray()
                }).ToArray();
            if (section == "all" || section == "inventory")
                result["inventory"] = new
                {
                    items = data.Items.Select((item, slot) => ItemInfo(item, slot)).ToArray(),
                    misc = data.Player.MiscItems.Select((value, id) => new { id, name = SafeName(() => Database.GetMiscItemName(id)), value }).ToArray(),
                    gifts = Enumerable.Range(0, Player_V23.COUNT_GIFT_ITEMS)
                        .Select(id => new { id, name = SafeName(() => Database.GetGiftItemName(id)), value = data.Player.GetGiftItem(id) }).ToArray()
                };
            if (section == "all" || section == "battalions")
                result["battalions"] = data.Player.Battalions.Select((battalion, slot) => new
                {
                    slot, battalion.CharacterId, battalion.Exp,
                    Stamina = save.GetBattalionEndurance(slot),
                    storedStamina = battalion.Stamina,
                    maximumEndurance = ObtainableBattalions.FullEndurance(battalion.Type),
                    battalion.Type, battalion.Skill,
                    name = SafeName(() => Database.GetBattalionName(battalion.Type))
                }).ToArray();
            if (section == "all" || section == "activities")
                result["activities"] = new
                {
                    data.Activities.Reputation, data.Activities.InstructExp,
                    data.Activities.ActivityExplore, data.Activities.ActivityLesson, data.Activities.ActivityBattle,
                    data.Activities.Statue1, data.Activities.Statue2, data.Activities.Statue3, data.Activities.Statue4,
                    playLog = new { data.Activities.PlayLog_Wark, data.Activities.PlayLog_Lecture,
                        data.Activities.PlayLog_ToBtl, data.Activities.PlayLog_Rest, data.Activities.PlayLog_Trnmnt,
                        data.Activities.PlayLog_Sing, data.Activities.PlayLog_Lunch, data.Activities.PlayLog_Cooking,
                        data.Activities.PlayLog_Drill, data.Activities.PlayLog_Teaparty, data.Activities.PlayLog_SCOUT },
                    quests = data.Activities.QuestStateList.Select((value, id) => new { id,
                        name = SafeName(() => Database.GetQuestName(id)), value }).ToArray()
                };
            if (section == "all" || section == "supports")
                result["supports"] = data.Player.CharacterSupportValues.Select((value, id) => new
                { id, name = SafeName(() => Database.GetSupportTalkName(id)), value,
                    rank = SupportPairRanks.RankForPoints(id, value),
                    maximumRank = SupportPairRanks.MaxRank(id) }).ToArray();
            if (result.Count == 3) throw new ArgumentException("Unknown section: " + section);
            return result;
        }

        private static object ItemInfo(Item item, int slot) => new
        {
            slot, item.Id, name = SafeName(() => Database.GetItemName(item.Id)), item.Durability, item.Amount
        };

        private static string SafeName(Func<string> getName)
        {
            try { return getName(); }
            catch { return Database.STR_UNKNOWN; }
        }

        private static object Catalog(string type)
        {
            switch (type.ToLowerInvariant())
            {
                case "items": return Database.ItemList.Select(entry => new { id = entry.Key, name = entry.Value }).ToArray();
                case "characters": return Database.UnitList.Select(entry => new
                {
                    id = entry.Key,
                    name = entry.Value,
                    mainCharacterId = entry.Key < 0 ? -1 : Database.BinaryDatabase.CharacterEntries[entry.Key].MainCharacterId
                }).ToArray();
                case "classes": return Database.ClassList.Select(entry => new { id = entry.Key, name = entry.Value }).ToArray();
                case "battalions": return Database.BattalionList.Select(entry => new { id = entry.Key, name = entry.Value }).ToArray();
                case "obtainable-battalions": return ObtainableBattalions.All.Select(entry => new
                {
                    id = entry.Type, name = Database.GetBattalionName(entry.Type),
                    experience = ObtainableBattalions.Experience, stamina = entry.Stamina, skill = entry.Skill
                }).ToArray();
                case "battalion-skills": return Database.BattalionSkillList.Select(entry => new { id = entry.Key, name = entry.Value }).ToArray();
                case "abilities": return Database.AbilityList.Select(entry => new { id = entry.Key, name = entry.Value }).ToArray();
                case "arts": return Database.CombatArtList.Select(entry => new { id = entry.Key, name = entry.Value }).ToArray();
                case "quests": return Enumerable.Range(0, 150).Select(id => new { id, name = SafeName(() => Database.GetQuestName(id)) }).ToArray();
                case "supports": return Enumerable.Range(0, Player_V23.COUNT_SUPPORT).Select(id => new
                {
                    id, name = SafeName(() => Database.GetSupportTalkName(id)),
                    ranks = SupportPairRanks.AvailableRanks(id), maximumRank = SupportPairRanks.MaxRank(id)
                }).ToArray();
                case "support-ranks": return SupportRankPresets.Values.Select(entry => new { rank = entry.Name, points = entry.Points }).ToArray();
                default: throw new ArgumentException("Unknown catalog: " + type);
            }
        }

        private static object CatalogDetail(string type, int id)
        {
            var database = Database.BinaryDatabase;
            switch (type.ToLowerInvariant())
            {
                case "characters" when id >= 0 && id < database.CharacterEntries.Count:
                    return new { id, name = Database.GetUnitName(id), details = database.CharacterEntries[id].GenerateDebugOut() };
                case "classes" when id >= 0 && id < database.ClassEntries.Count:
                    return new { id, name = Database.GetClassName(id), details = database.ClassEntries[id].GenerateDebugOut() };
                case "items" when database.ItemEntries.TryGetValue(id, out var item):
                    return new { id, name = Database.GetItemName(id), details = item.GenerateDebugOut() };
                default: throw new ArgumentException($"No database detail for {type} ID {id}.");
            }
        }

        private static void Print(object value) => Console.WriteLine(Json.Serialize(value));
    }
}
