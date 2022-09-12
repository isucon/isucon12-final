using System;

public static class TextUtil
{
    public static string FormatDateFromUnixTime(long unixTime)
    {
        return DateTimeOffset.FromUnixTimeSeconds(unixTime).ToString("yyyy/MM/dd");
    }
    
    public static string FormatShortText(long num)
    {
        if (num >= 1000 * 1000 * 1000)
        {
            return (float)num / 1000 / 1000 / 1000 + "G";
        }
        if (num >= 1000 * 1000)
        {
            return (float)num / 1000 / 1000 + "M";
        }
        if (num >= 1000)
        {
            return (float)num / 1000 + "K";
        }

        return num.ToString();
    }
}
