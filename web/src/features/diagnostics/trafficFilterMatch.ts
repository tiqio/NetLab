export interface TrafficFilterMatch extends Record<string, unknown> {
  source_address?: string;
  destination_address?: string;
  protocol?: string;
  source_port?: number;
  destination_port?: number;
  and?: TrafficFilterMatch[];
  or?: TrafficFilterMatch[];
  not?: TrafficFilterMatch;
}

export function parseTrafficFilterMatch(value: string): TrafficFilterMatch {
  const expression = value.trim();
  if (!expression) throw new Error("Enter a traffic characteristic");
  const compiledBidirectionalPort = expression.match(
    /^\((tcp|udp)\s+and\s+src\s+port\s+(\d+)\)\s+or\s+\(\1\s+and\s+dst\s+port\s+\2\)$/i,
  );
  if (compiledBidirectionalPort) {
    const protocol = compiledBidirectionalPort[1].toLowerCase();
    const port = Number(compiledBidirectionalPort[2]);
    return {
      or: [
        { protocol, source_port: port },
        { protocol, destination_port: port },
      ],
    };
  }
  if (/\b(or|not)\b/i.test(expression))
    throw new Error(
      "OR and NOT are not supported by the text field yet; use one characteristic or the API/MCP structured match.",
    );
  const match: TrafficFilterMatch = {};
  const sourceAddress = expression.match(
    /\bsrc\s+(?:(?:host|net)\s+)?(?!port\b)([^\s]+)/i,
  );
  const destinationAddress = expression.match(
    /\bdst\s+(?:(?:host|net)\s+)?(?!port\b)([^\s]+)/i,
  );
  const sourcePort = expression.match(/\bsrc\s+port\s+(\d+)/i);
  const destinationPort = expression.match(/\bdst\s+port\s+(\d+)/i);
  const genericPort = expression.match(/\bport\s+(\d+)/i);
  const protocol = expression.match(/\b(tcp|udp|icmp6?|arp|ip6?)\b/i);
  if (sourceAddress) match.source_address = sourceAddress[1];
  if (destinationAddress) match.destination_address = destinationAddress[1];
  if (protocol) match.protocol = protocol[1].toLowerCase();
  if (sourcePort) match.source_port = Number(sourcePort[1]);
  if (destinationPort) match.destination_port = Number(destinationPort[1]);
  if (
    (sourcePort || destinationPort) &&
    !["tcp", "udp"].includes(match.protocol || "")
  )
    throw new Error("Source and destination ports require tcp or udp");
  if (!sourcePort && !destinationPort && genericPort) {
    const port = Number(genericPort[1]);
    const protocols = match.protocol ? [match.protocol] : ["tcp", "udp"];
    if (protocols.some((item) => !["tcp", "udp"].includes(item)))
      throw new Error("A generic port requires tcp or udp");
    const base = { ...match };
    delete base.protocol;
    return {
      or: protocols.flatMap((item) => [
        { ...base, protocol: item, source_port: port },
        { ...base, protocol: item, destination_port: port },
      ]),
    };
  }
  if (!Object.keys(match).length)
    throw new Error("No supported traffic characteristic was found");
  return match;
}
