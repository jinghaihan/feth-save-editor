using System;
using System.Linq;

namespace FethEditor.Core
{
    public static class SupportRankPresets
    {
        public static readonly (string Name, int Points)[] Values =
        {
            ("None", 0), ("C", 101), ("C+", 201), ("B", 301),
            ("B+", 451), ("A", 601), ("A+", 801), ("S", 1001)
        };

        public static int PointsFor(string name)
        {
            foreach (var preset in Values)
                if (string.Equals(preset.Name, name, StringComparison.OrdinalIgnoreCase))
                    return preset.Points;
            throw new ArgumentException("Unknown support rank: " + name, nameof(name));
        }

        public static string NameFor(int points) =>
            Values.Last(preset => points >= preset.Points).Name;
    }
}
